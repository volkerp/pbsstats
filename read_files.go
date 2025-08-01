package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type FileInfo struct {
	refChunks    []uint32 // list of referred chunks, index into the digest index
	uniqueChunks uint32   // number of unique chunks
}

func NewFileInfo() *FileInfo {
	return &FileInfo{
		refChunks:    make([]uint32, 0),
		uniqueChunks: 0,
	}
}

// calculateFileDedup for single file
func calculateFileDedup(fileInfo *FileInfo) {
	// Calculate unique chunks
	unique := make(map[uint32]struct{})
	for _, idx := range fileInfo.refChunks {
		unique[idx] = struct{}{}
	}
	fileInfo.uniqueChunks = uint32(len(unique))
}

type DigestMap struct {
	digestIndex    map[Digest]uint32
	digestsByIndex map[uint32]Digest
	refCounter     map[uint32]int
	accuCounter    [256 * 256]uint // count digests beginning with the same 2 bytes 0x0000-0xffff
	accuRefCounter [256 * 256]uint // count references beginning with the same 2 bytes 0x0000-0xffff
}

func NewDigestMap() *DigestMap {
	return &DigestMap{
		digestIndex:    make(map[Digest]uint32),
		digestsByIndex: make(map[uint32]Digest),
		refCounter:     make(map[uint32]int),
		accuCounter:    [256 * 256]uint{},
		accuRefCounter: [256 * 256]uint{},
	}
}

type Files map[string]FileInfo

var (
	globalDataMutex        sync.RWMutex
	globalDigestsMap       DigestMap
	globalFileIndex        Files
	globalScanProgressChan chan string // Channel for scan progress updates
)

func (d *DigestMap) add(digest Digest) uint32 {
	var index uint32
	if _, exists := d.digestIndex[digest]; !exists {
		// digest not yet in index
		index = uint32(len(d.digestIndex))
		d.digestIndex[digest] = index
		d.digestsByIndex[index] = digest
		d.refCounter[index] = 1
		d.accuCounter[uint16(digest[0])<<8|uint16(digest[1])]++
	} else {
		index = d.digestIndex[digest]
		d.refCounter[index]++
	}
	d.accuRefCounter[uint16(digest[0])<<8|uint16(digest[1])]++

	return index
}

func (f *Files) addFileRef(filename string, digestIndex uint32) {
	if _, exists := (*f)[filename]; !exists {
		(*f)[filename] = *NewFileInfo()
	}
	fileInfo := (*f)[filename]
	fileInfo.refChunks = append(fileInfo.refChunks, digestIndex)
	(*f)[filename] = fileInfo
}

func calculateFilesDedup(files Files) Files {
	for filename, fileInfo := range files {
		// Calculate unique chunks
		unique := make(map[uint32]struct{})
		for _, idx := range fileInfo.refChunks {
			unique[idx] = struct{}{}
		}
		fileInfo.uniqueChunks = uint32(len(unique))
		files[filename] = fileInfo
	}
	return files
}

func scanIndexFiles(root string) error {
	fileChan := make(chan string, 100)              // Channel to pass file paths
	globalScanProgressChan = make(chan string, 100) // Channel for scan progress updates
	wg := &sync.WaitGroup{}                         // WaitGroup to wait for all workers

	// Worker function to process files
	worker := func() {
		defer wg.Done()
		for path := range fileChan {
			if filepath.Ext(path) == ".didx" {
				didx, err := readDidxFile(path)
				if err != nil {
					fmt.Printf("Error processing %s: %v\n", path, err)
					continue
				}

				globalDataMutex.Lock()
				fmt.Printf("\r\033[KProcessing file: %s", path)
				if len(globalScanProgressChan) >= 100 {
					<-globalScanProgressChan
				}
				globalScanProgressChan <- fmt.Sprintf("%d %s", len(globalFileIndex), path)
				for _, digest := range didx.Digests {
					digestIndex := globalDigestsMap.add(digest.Digest)
					globalFileIndex.addFileRef(path, digestIndex)
				}
				// Calculate unique chunks for the file
				fileInfo := globalFileIndex[path]
				calculateFileDedup(&fileInfo)
				globalFileIndex[path] = fileInfo
				globalDataMutex.Unlock()
			} else if filepath.Ext(path) == ".fidx" {
				fidx, err := readFidxFile(path)
				if err != nil {
					fmt.Printf("Error processing %s: %v\n", path, err)
					continue
				}

				globalDataMutex.Lock()
				fmt.Printf("\r\033[KProcessing file: %s", path)
				if len(globalScanProgressChan) >= 100 {
					<-globalScanProgressChan
				}
				globalScanProgressChan <- fmt.Sprintf("%d %s", len(globalFileIndex), path)
				for _, digest := range fidx.Digests {
					digestIndex := globalDigestsMap.add(digest)
					globalFileIndex.addFileRef(path, digestIndex)
				}
				// Calculate unique chunks for the file
				fileInfo := globalFileIndex[path]
				calculateFileDedup(&fileInfo)
				globalFileIndex[path] = fileInfo
				globalDataMutex.Unlock()
			}
		}
	}

	// Start worker goroutines
	numWorkers := 4 // Adjust based on your system's capabilities
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker()
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && filepath.Base(path) == ".chunks" {
			return filepath.SkipDir // Skip the ".chunks" directory
		}
		if !d.IsDir() && (filepath.Ext(path) == ".fidx" || filepath.Ext(path) == ".didx") {
			fileChan <- path // Send file path to the channel
		}
		return nil // Continue walking
	})

	close(fileChan) // Close the channel to signal workers to stop
	wg.Wait()       // Wait for all workers to finish
	close(globalScanProgressChan)
	fmt.Println("\n\033[KProcessed all files.")
	return err
}

func printOccurrences(digestMap *DigestMap, topN int) {
	fmt.Printf("Total unique digests/chunks: %d\n", len(digestMap.digestIndex))
	fmt.Printf("Top %d digest occurrences in indices:\n", topN)
	type digestCount struct {
		digest Digest
		count  int
	}

	var digestList []digestCount
	for digestIdx, count := range digestMap.refCounter {
		digest := digestMap.digestsByIndex[digestIdx]
		digestList = append(digestList, digestCount{digest, count})
	}

	// Sort by count in descending order
	sort.Slice(digestList, func(i, j int) bool {
		return digestList[i].count > digestList[j].count
	})

	// Print the top N entries
	for i, entry := range digestList {
		if i >= topN {
			break
		}
		fmt.Printf("%032x: %d\n", entry.digest, entry.count)
	}
}

func printFileDigestRefs(fileIndex Files) {
	fmt.Println("File references for each digest:")
	for filename, fileInfo := range fileIndex {
		fmt.Printf("%s: ", filename)
		for _, ref := range fileInfo.refChunks {
			fmt.Printf("%d ", ref)
		}
		fmt.Println()
	}
}

func printFileDedupHighest(fileIndex Files, topN int) {
	fmt.Printf("Top %d highest dedup ratio files:\n", topN)
	type fileDedup struct {
		filename string
		dedup    float32
	}
	var fileDedupList []fileDedup

	for filename, fileInfo := range fileIndex {
		dedup := float32(len(fileInfo.refChunks)) / float32(fileInfo.uniqueChunks)
		fileDedupList = append(fileDedupList, fileDedup{filename, dedup})
	}
	// Sort by ratio in descending order
	sort.Slice(fileDedupList, func(i, j int) bool {
		return fileDedupList[i].dedup > fileDedupList[j].dedup
	})

	// Print the top N entries
	for i, entry := range fileDedupList {
		if i >= topN {
			break
		}
		fmt.Printf("%s: %.2f\n", entry.filename, entry.dedup)
	}

}
