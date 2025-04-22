package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type DigestCounter map[Digest]int

func scanAndCountDigests(root string) (DigestCounter, error) {
	counter := make(DigestCounter)
	counterMutex := &sync.Mutex{}      // To safely update the counter map
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
			counterMutex.Lock()
			for _, digest := range fidx.Digests {
				counter[digest]++
			}
			counterMutex.Unlock()
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

	return counter, err
}

func printOccurrences(counter DigestCounter, topN int) {
	fmt.Printf("Top %d Digest occurrences:\n", topN)
	type digestCount struct {
		digest Digest
		count  int
	}

	var digestList []digestCount
	for digest, count := range counter {
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
		fmt.Printf("%x: %d\n", entry.digest, entry.count)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: scan_fidx <directory>")
		os.Exit(1)
	}
	root := os.Args[1]
	counter, err := scanAndCountDigests(root)
	if err != nil {
		fmt.Printf("Scan error: %v\n", err)
		os.Exit(1)
	}

	printOccurrences(counter, 50)
}
