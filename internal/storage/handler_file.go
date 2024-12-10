package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
)

// routeHandler routes the HTTP request to the appropriate handler based on the request method.
func (s *Storage) routeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		s.uploadHandler(w, r)
	case http.MethodGet:
		s.downloadHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// uploadHandler handles the upload of a file, calculates its SHA-256 hash, and stores it temporarily.
func (s *Storage) uploadHandler(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(r.URL.Path)
	if filename == "" || filename != "segment" {
		http.Error(w, "invalid filename", http.StatusBadRequest)
		return
	}

	tmpFile, err := os.CreateTemp(s.Dir, "*.tmp")
	if err != nil {
		log.Printf("Storage %s uploadHandler create temp file error: %v", s.Addr, r.URL.Path)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hasher := sha256.New()
	_, err = io.Copy(tmpFile, io.TeeReader(r.Body, hasher))
	if err != nil && err != io.EOF {
		log.Printf("Storage %s uploadHandler: %s\n\tFILE: %s\n FAILED", s.Addr, r.URL.Path, tmpFile.Name())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return
	}
	tmpFile.Close()

	hash := hex.EncodeToString(hasher.Sum(nil))
	log.Printf("Storage %s uploadHandler: %s\n\tFILE: %s\n\tHASH: %s OK", s.Addr, r.URL.Path, tmpFile.Name(), hash)

	w.Header().Set("X-Hash", hash)
	w.Header().Set("X-Filename", tmpFile.Name())

	atomic.AddInt64(&s.Used, r.ContentLength)
}

// downloadHandler serves the requested file from the storage directory.
func (s *Storage) downloadHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Storage %s downloadHandler: %s", s.Addr, r.URL.Path)

	filename := filepath.Base(r.URL.Path)
	if filename == "" {
		http.Error(w, "filename is required", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(s.Dir, filename)
	http.ServeFile(w, r, filePath)
}
