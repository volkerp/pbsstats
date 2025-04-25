package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func apiStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	globalDataMutex.RLock()
	defer globalDataMutex.RUnlock()
	type digestEntry struct {
		//Digest string    `json:"digest"`
		DigestIndex uint32 `json:"digest_index"`
		Count       int    `json:"count"`
	}
	var digests []digestEntry
	for idx := range globalDigestsMap.digestsByIndex {
		count := globalDigestsMap.counter[idx]
		digests = append(digests, digestEntry{
			//Digest: fmt.Sprintf("%032x", digest),
			DigestIndex: idx,
			Count:       count,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(digests)
}

func apiFilesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	globalDataMutex.RLock()
	defer globalDataMutex.RUnlock()

	type fileEntry struct {
		Filename     string `json:"filename"`
		NumRefChunks int    `json:"num_ref_chunks"`
		UniqueChunks uint32 `json:"unique_chunks"`
	}
	var files []fileEntry
	for filename, info := range globalFileIndex {
		files = append(files, fileEntry{
			Filename:     filename,
			NumRefChunks: len(info.refChunks),
			UniqueChunks: info.uniqueChunks,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func apiFileRefChunksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	filename := r.URL.Query().Get("filename")
	if filename == "" {
		http.Error(w, "filename parameter required", http.StatusBadRequest)
		return
	}

	globalDataMutex.RLock()
	defer globalDataMutex.RUnlock()
	info, ok := globalFileIndex[filename]
	if !ok {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	type refChunksResponse struct {
		RefChunks []uint32 `json:"ref_chunks"`
	}
	response := refChunksResponse{
		RefChunks: info.refChunks,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func startWebServer(port int) {
	http.HandleFunc("/api/stats", apiStatsHandler)
	http.HandleFunc("/api/digests", apiDigestsHandler)
	http.HandleFunc("/api/files", apiFilesHandler)
	http.HandleFunc("/api/refchunks", apiFileRefChunksHandler)
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
