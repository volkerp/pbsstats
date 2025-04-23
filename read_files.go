package main

import (
	"flag"
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

type DigestMap struct {
	digestIndex    map[Digest]uint32
	digestsByIndex map[uint32]Digest
	counter        map[uint32]int
}

func NewDigestMap() *DigestMap {
	return &DigestMap{
		digestIndex:    make(map[Digest]uint32),
		digestsByIndex: make(map[uint32]Digest),
		counter:        make(map[uint32]int),
	}
}

func (d *DigestMap) add(digest Digest) uint32 {
	if _, exists := d.digestIndex[digest]; !exists {
		index := uint32(len(d.digestIndex))
		d.digestIndex[digest] = index
		d.digestsByIndex[index] = digest
		d.counter[index] = 1
		return index
	} else {
		index := d.digestIndex[digest]
		d.counter[index]++
		return index
	}
}

type Files map[string]FileInfo

func (f *Files) addFileRef(filename string, digestIndex uint32) {
	if _, exists := (*f)[filename]; !exists {
		(*f)[filename] = *NewFileInfo()
	}
	fileInfo := (*f)[filename]
	fileInfo.refChunks = append(fileInfo.refChunks, digestIndex)
	(*f)[filename] = fileInfo
}

func calculateFileDedup(files Files) Files {
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

func scanIndexFiles(root string) (*DigestMap, Files, error) {
	digestsMap := NewDigestMap()
	fileIndex := make(Files)
	digestsIndexMutex := &sync.Mutex{} // To safely update the counter map
	fileChan := make(chan string, 100) // Channel to pass file paths
	wg := &sync.WaitGroup{}            // WaitGroup to wait for all workers

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

				digestsIndexMutex.Lock()
				fmt.Printf("Processing file: %s\n", path)
				for _, digest := range didx.Digests {
					digestIndex := digestsMap.add(digest.Digest)
					fileIndex.addFileRef(path, digestIndex)
				}
				digestsIndexMutex.Unlock()
			} else if filepath.Ext(path) == ".fidx" {
				fidx, err := readFidxFile(path)
				if err != nil {
					fmt.Printf("Error processing %s: %v\n", path, err)
					continue
				}

				digestsIndexMutex.Lock()
				fmt.Printf("Processing file: %s\n", path)
				for _, digest := range fidx.Digests {
					digestIndex := digestsMap.add(digest)
					fileIndex.addFileRef(path, digestIndex)
				}
				digestsIndexMutex.Unlock()
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

	return digestsMap, fileIndex, err
}

func printOccurrences(digestMap *DigestMap, topN int) {
	fmt.Printf("Top %d Digest occurrences:\n", topN)
	type digestCount struct {
		digest Digest
		count  int
	}

	var digestList []digestCount
	for digestIdx, count := range digestMap.counter {
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
		digestIndex := digestMap.digestIndex[entry.digest]
		fmt.Printf("%x (%d): %d\n", entry.digest, digestIndex, entry.count)
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

func main() {
	var topChunks int
	var topFiles int

	flag.IntVar(&topChunks, "top-chunks", 50, "Show top N most referenced chunks")
	flag.IntVar(&topFiles, "top-files", 50, "Show top N files with highest dedup ratio")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: scan_fidx [--top-chunks N] [--top-files N] <directory>")
		os.Exit(1)
	}
	root := flag.Arg(0)

	digestsMap, fileIndex, err := scanIndexFiles(root)
	if err != nil {
		fmt.Printf("Scan error: %v\n", err)
		os.Exit(1)
	}

	fileIndex = calculateFileDedup(fileIndex)

	printOccurrences(digestsMap, topChunks)
	printFileDedupHighest(fileIndex, topFiles)
	fmt.Printf("Total unique digests: %d\n", len(digestsMap.digestIndex))
}
