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

type DigestIndexMap struct {
	index    []Digest       // Slice to maintain order and index access
	counter  map[Digest]int // Map for fast lookup
	fileInfo map[string]FileInfo
}

// NewDigestIndexMap creates and initializes a new DigestIndexMap
func NewDigestIndexMap() *DigestIndexMap {
	return &DigestIndexMap{
		index:    make([]Digest, 0),
		counter:  make(map[Digest]int),
		fileInfo: make(map[string]FileInfo),
	}
}

// get the index of a digest
func (d *DigestIndexMap) indexOfDigest(digest Digest) int {
	for i, v := range d.index {
		if v == digest {
			return i
		}
	}
	return -1
}

// add digest to the index
func (d *DigestIndexMap) add(digest Digest) {
	if _, exists := d.counter[digest]; !exists {
		d.index = append(d.index, digest)
	}
	d.counter[digest]++
}

// add or update the file reference
func (d *DigestIndexMap) addFileRef(filename string, digest Digest) {
	digestIndex := d.indexOfDigest(digest)
	if digestIndex == -1 {
		fmt.Printf("Digest %x not found in index\n", digest)
		return
	}
	if _, exists := d.fileInfo[filename]; !exists {
		d.fileInfo[filename] = *NewFileInfo()
	}
	fileInfo := d.fileInfo[filename]
	fileInfo.refChunks = append(fileInfo.refChunks, uint32(digestIndex))
	d.fileInfo[filename] = fileInfo
}

func calculateFileDedup(digestsIndex *DigestIndexMap) int {
	dedupCount := 0
	for _, count := range digestsIndex.counter {
		if count > 1 {
			dedupCount += count - 1
		}
	}
	return dedupCount
}

func scanAndCountDigests(root string) (*DigestIndexMap, error) {
	digestsIndex := NewDigestIndexMap()
	digestsIndexMutex := &sync.Mutex{} // To safely update the counter map
	fileChan := make(chan string, 100) // Channel to pass file paths
	wg := &sync.WaitGroup{}            // WaitGroup to wait for all workers

	// Worker function to process files
	worker := func() {
		defer wg.Done()
		for path := range fileChan {
			fidx, err := readFidxFile(path)
			if err != nil {
				fmt.Printf("Error processing %s: %v\n", path, err)
				continue
			}
			// Safely update the counter
			digestsIndexMutex.Lock()
			for _, digest := range fidx.Digests {
				digestsIndex.add(digest)
				digestsIndex.addFileRef(path, digest)
			}
			digestsIndexMutex.Unlock()
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
		if !d.IsDir() && filepath.Ext(path) == ".fidx" {
			fmt.Printf("Processing file: %s\n", path)
			fileChan <- path // Send file path to the channel
		}
		return nil // Continue walking
	})

	close(fileChan) // Close the channel to signal workers to stop
	wg.Wait()       // Wait for all workers to finish

	return digestsIndex, err
}

func printOccurrences(digestIndex *DigestIndexMap, topN int) {
	fmt.Printf("Top %d Digest occurrences:\n", topN)
	type digestCount struct {
		digest Digest
		count  int
	}

	var digestList []digestCount
	for digest, count := range digestIndex.counter {
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
		digestIndex := digestIndex.indexOfDigest(entry.digest)
		fmt.Printf("%x (%d): %d\n", entry.digest, digestIndex, entry.count)
	}
}

func printDigestRefs(digestsIndex *DigestIndexMap) {
	fmt.Println("File references for each digest:")
	for filename, fileInfo := range digestsIndex.fileInfo {
		fmt.Printf("%s: ", filename)
		for _, ref := range fileInfo.refChunks {
			fmt.Printf("%d ", ref)
		}
		fmt.Println()
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: scan_fidx <directory>")
		os.Exit(1)
	}
	root := os.Args[1]
	digestIndex, err := scanAndCountDigests(root)
	if err != nil {
		fmt.Printf("Scan error: %v\n", err)
		os.Exit(1)
	}

	printOccurrences(digestIndex, 50)
	printDigestRefs(digestIndex)
}
