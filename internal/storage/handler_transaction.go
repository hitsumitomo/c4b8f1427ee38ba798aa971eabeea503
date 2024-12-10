package storage

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
)

// rollbackHandler handles DELETE requests to rollback a transaction by removing a specified file.
func (s *Storage) rollbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filePath := r.Header.Get("X-Filename")
	if filePath == "" {
		log.Printf("Storage %s rollbackHandler error: X-Filename is required", s.Addr)
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	size := info.Size()

	log.Printf("Storage %s rollbackHandler: %s\n\tREMOVE: %v size: %v", s.Addr, r.URL.Path, filePath, size)
	if err := os.Remove(filePath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	atomic.AddInt64(&s.Used, -size)
}

// commitHandler handles POST requests to commit a transaction by renaming a temporary file to its final destination.
func (s *Storage) commitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file := filepath.Base(r.URL.Path)
	tmpFilePath := r.Header.Get("X-Filename")

	if file == "" || tmpFilePath == "" {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(s.Dir, file)

	log.Printf("Storage %s commitHandler: %s\n\tFILE: %s\n\tRENAME: %s", s.Addr, r.URL.Path, tmpFilePath, filePath)
	if err := os.Rename(tmpFilePath, filePath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}







