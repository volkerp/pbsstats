package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var topChunks int
	var topFiles int
	var webPort int

	flag.IntVar(&topChunks, "top-chunks", 50, "Show top N most referenced chunks")
	flag.IntVar(&topFiles, "top-files", 50, "Show top N files with highest dedup ratio")
	flag.IntVar(&webPort, "web-port", 8080, "Start webserver on given port (0 disables webserver)")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: scan_fidx [--top-chunks N] [--top-files N] [--web-port PORT] <directory>")
		os.Exit(1)
	}
	root := flag.Arg(0)

	// Initialize global maps
	globalDigestsMap = *NewDigestMap()
	globalFileIndex = make(Files)

	if webPort != 0 {
		startWebServer(webPort)
	}

	err := scanIndexFiles(root)
	if err != nil {
		fmt.Printf("Scan error: %v\n", err)
		os.Exit(1)
	}

	printOccurrences(&globalDigestsMap, topChunks)
	printFileDedupHighest(globalFileIndex, topFiles)

	if webPort != 0 {
		// Block forever to keep the webserver running
		select {}
	}
}
