package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
)

// corsMiddleware wraps an http.HandlerFunc to set CORS headers once for all handlers.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

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
		//Digest string    `json:"digest"`
		DigestIndex uint32 `json:"digest_index"`
		Count       int    `json:"count"`
	}
	var digests []digestEntry
	for idx := range globalDigestsMap.digestsByIndex {
		count := globalDigestsMap.refCounter[idx]
		digests = append(digests, digestEntry{
			//Digest: fmt.Sprintf("%032x", digest),
			DigestIndex: idx,
			Count:       count,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(digests)
}

func apiDigestsHandler2(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	globalDataMutex.RLock()
	defer globalDataMutex.RUnlock()
	type digestEntry struct {
		Digest      string `json:"digest"`
		DigestIndex uint32 `json:"digest_index"`
		Count       int    `json:"count"`
	}
	var digests []digestEntry
	for idx, digest := range globalDigestsMap.digestsByIndex {
		digestHex := fmt.Sprintf("%032x", digest)
		if prefix == "" || (len(prefix) <= len(digestHex) && digestHex[:len(prefix)] == prefix) {
			count := globalDigestsMap.refCounter[idx]
			digests = append(digests, digestEntry{
				Digest:      digestHex,
				DigestIndex: idx,
				Count:       count,
			})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(digests)
}

func apiFilesHandler(w http.ResponseWriter, r *http.Request) {
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

// apiFileRefChunksHandler handles HTTP requests for retrieving the reference chunk IDs of a specified file.
func apiFileRefChunksHandler(w http.ResponseWriter, r *http.Request) {
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

func apiAccuCounterHandler(w http.ResponseWriter, r *http.Request) {
	globalDataMutex.RLock()
	defer globalDataMutex.RUnlock()
	accuCount := struct {
		AccuCount    [256 * 256]uint `json:"accu_count"`
		AccuRefCount [256 * 256]uint `json:"accu_ref_count"`
	}{
		AccuCount:    globalDigestsMap.accuCounter,
		AccuRefCount: globalDigestsMap.accuRefCounter,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accuCount)
}

func spaHandler(w http.ResponseWriter, r *http.Request) {
	// handle requests to /
	if r.URL.Path == "/" || r.URL.Path == "" || r.URL.Path == "index.html" || r.URL.Path == "/index.html" {
		// Open the embedded index.html
		indexFile, _ := embeddedFiles.Open("index.html")
		defer indexFile.Close()

		index, _ := io.ReadAll(indexFile)
		w.WriteHeader(http.StatusAccepted)
		w.Write(index)
		return
	} else {
		// Serve static files from the embedded file system
		spaFileServer.ServeHTTP(w, r)
	}
}

//go:embed web/dist
var embeddedFilesWebDist embed.FS // embedded file system prefixed with "web/dist"

var embeddedFiles fs.FS // embedded file withoud prefix "web/dist"

var spaFileServer http.Handler

//go:generate echo "Building frontend assets..."
//go:generate npm run build --prefix web

func startWebServer(port int) {
	// embeddedFiles is the embedded file system containing the static files
	// get rid of prefix "web/dist/"
	embeddedFiles, _ = fs.Sub(embeddedFilesWebDist, "web/dist")

	// http.FS converts the fs.FS (our subFS) to the http.FileSystem interface
	// http.FileServer creates a handler that serves HTTP requests
	// with the contents of the file system (http.FS(subFS)).
	spaFileServer = http.FileServer(http.FS(embeddedFiles))

	http.HandleFunc("/", corsMiddleware(spaHandler))
	http.HandleFunc("/api/stats", corsMiddleware(apiStatsHandler))
	http.HandleFunc("/api/digests", corsMiddleware(apiDigestsHandler))
	http.HandleFunc("/api/chunks", corsMiddleware(apiDigestsHandler2))
	http.HandleFunc("/api/files", corsMiddleware(apiFilesHandler))
	http.HandleFunc("/api/refchunks", corsMiddleware(apiFileRefChunksHandler))
	http.HandleFunc("/api/accucounter", corsMiddleware(apiAccuCounterHandler))
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
