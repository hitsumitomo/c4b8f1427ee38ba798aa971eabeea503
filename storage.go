package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

type Storage struct {
	Limit       int64
	Used        int64
	Addr        string
	Dir         string
	RegisterURL string
	Registered  time.Time
}

func (s *Storage) rollbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filePath := r.Header.Get("Filename")
	if filePath == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	size := info.Size()

	log.Printf("Addr: %s rollbackHandler:\n\t%s: %s\n\tREMOVE: %v size: %v", s.Addr, r.Method, r.URL.Path, filePath, size)
	if err := os.Remove(filePath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	atomic.AddInt64(&s.Used, -size)
}

func (s *Storage) commitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file := filepath.Base(r.URL.Path)
	tmpFilePath := r.Header.Get("Filename")

	if file == "" || tmpFilePath == "" {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(s.Dir, file)

	log.Printf("Addr: %s commitHandler:\n\t%s: %s\n\tTEMP: %s\n\tRENAMED: %s", s.Addr, r.Method, r.URL.Path, tmpFilePath, filePath)
	if err := os.Rename(tmpFilePath, filePath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Storage) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := filepath.Base(r.URL.Path)
	if filename == "" || filename != "segment" {
		http.Error(w, "invalid filename", http.StatusBadRequest)
		return
	}

	tmpFile, err := os.CreateTemp(s.Dir, "*.tmp")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hasher := sha256.New()
	_, err = io.Copy(tmpFile, io.TeeReader(r.Body, hasher))
	if err != nil && err != io.EOF {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return
	}
	tmpFile.Close()

	hash := hex.EncodeToString(hasher.Sum(nil))
	log.Printf("Addr: %s uploadHandler:\n\t%s: %s\n\tTEMP: %s\n\tHASH: %s", s.Addr, r.Method, r.URL.Path, tmpFile.Name(), hash)

	w.Header().Set("X-Hash", hash)
	w.Header().Set("Filename", tmpFile.Name())

	atomic.AddInt64(&s.Used, r.ContentLength)
}

func (s *Storage) downloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Addr: %s, downloadHandler: %s %s", s.Addr, r.Method, r.URL.Path)

	filename := filepath.Base(r.URL.Path)
	if filename == "" {
		http.Error(w, "filename is required", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(s.Dir, filename)
	http.ServeFile(w, r, filePath)
}

func (s *Storage) register() error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("POST", s.RegisterURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("register", "true")
	req.Header.Set("addr", s.Addr)
	req.Header.Set("limit", strconv.FormatInt(s.Limit, 10))
	req.Header.Set("used", strconv.FormatInt(s.Used, 10))

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Addr: %s, Failed to register storage: %v", s.Addr, err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Register failed with status %v", resp.StatusCode)
	}
	return nil
}

func (s *Storage) healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Storage server is healthy\nCapacity: %d\nUsed: %d\n", s.Limit, s.Used)
}

func main() {
	s := &Storage{
		Limit: 10 * 1024 * 1024 * 1024, // 10 GB
	}

	s.Addr = os.Getenv("STORAGE_ADDR")
	s.Dir = os.Getenv("STORAGE_DIR")
	s.RegisterURL = os.Getenv("REGISTER_URL")

	log.Printf("Addr: %s, Dir: %s, RegisterURL: %s", s.Addr, s.Dir, s.RegisterURL)

	if s.Addr == "" || s.Dir == "" || s.RegisterURL == "" {
		log.Fatal("STORAGE_ADDR, STORAGE_DIR and REGISTER_URL must be set")
	}

	if err := os.MkdirAll(s.Dir, 0755); err != nil {
		log.Fatalf("Addr: %s, Failed to create storage directory: %v", s.Addr, err)
	}

	var total int64
	err := filepath.Walk(s.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if strings.HasSuffix(info.Name(), ".tmp") {
				os.Remove(path)
				return nil
			}
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Addr: %s, Failed to scan storage directory: %v", s.Addr, err)
	}

	if total > s.Limit {
		log.Fatalf("Addr: %s, Storage limit exceeded: %v > %v", s.Addr, total, s.Limit)
		s.Used = s.Limit
	} else {
		s.Used = total
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/upload/", s.uploadHandler)
	mux.HandleFunc("/download/", s.downloadHandler)
	mux.HandleFunc("/rollback/", s.rollbackHandler)
	mux.HandleFunc("/commit/", s.commitHandler)
	mux.HandleFunc("/health", s.healthHandler)

	server := &http.Server{
		Addr:    s.Addr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Addr: %s, Failed to start server: %v", server.Addr, err)
		}
	}()

	time.Sleep(500 * time.Millisecond)
	log.Printf("Storage started at %v", server.Addr)

	// Register storage
	for {
		if err := s.register(); err != nil {
			log.Printf("Addr: %s, Failed to register storage: %v", server.Addr, err)
			time.Sleep(time.Second)
			continue
		}
		break
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGHUP)

	for {
		select {
		case sig := <-signalChan:
			if sig == syscall.SIGHUP {
				log.Printf("Addr: %s, Received SIGHUP signal, reloading configuration", server.Addr)
			}
		}
	}
}
