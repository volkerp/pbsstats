package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func apiStatsHandler(w http.ResponseWriter, r *http.Request) {
	globalDataMutex.RLock()
	defer globalDataMutex.RUnlock()
	stats := struct {
		TotalUniqueDigests int `json:"total_unique_digests"`
		TotalFiles         int `json:"total_files"`
	}{
		TotalUniqueDigests: len(globalDigestsMap.digestIndex),
		TotalFiles:         len(globalFileIndex),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func apiDigestsHandler(w http.ResponseWriter, r *http.Request) {
	globalDataMutex.RLock()
	defer globalDataMutex.RUnlock()
	type digestEntry struct {
		Digest string `json:"digest"`
		Count  int    `json:"count"`
	}
	var digests []digestEntry
	for idx, digest := range globalDigestsMap.digestsByIndex {
		count := globalDigestsMap.counter[idx]
		digests = append(digests, digestEntry{
			Digest: fmt.Sprintf("%032x", digest),
			Count:  count,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(digests)
}

func startWebServer(port int) {
	http.HandleFunc("/api/stats", apiStatsHandler)
	http.HandleFunc("/api/digests", apiDigestsHandler)
	go func() {
		addr := fmt.Sprintf(":%d", port)
		fmt.Printf("Starting webserver on %s (API at /api/)\n", addr)
		fmt.Println("API for digests at /api/digests")
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			fmt.Printf("Webserver error: %v\n", err)
		}
	}()
}
