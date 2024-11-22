package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

var (
	MongoURL    string
	ManagerAddr string

	ErrAlreadyExist = errors.New("File already exists")
)

const (
	chunkSize  = 4096
	unit       = 1024
	unitChar   = "KMGTPE"
	timeout    = 10 * time.Second
	storedMark = "[STORED]"
)

type FileInfo struct {
	Name     string   `json:"name"`
	Hash     string   `json:"hash"`
	Size     int64    `json:"size,omitempty"`
	Metadata []string `json:"metadata,omitempty"`
}

type Storage struct {
	Limit int
	Used  int
	URL   string
	free  int
}

type StorageManager struct {
	sync.RWMutex
	storages   []*Storage
	registered map[string]time.Time
	server     *http.Server
	mongodb    *MongoDB
}

type Scheme struct {
	url     string
	size    int
	tmpfile string
}

type chunkResult struct {
	index int
	url   string
	err   error
}

// NewStorageManager creates a new storage manager.
func NewStorageManager() *StorageManager {
	mongo, err := NewMongoDB(MongoURL, "storage")
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	log.Printf("Connected to MongoDB: %s\n", MongoURL)

	sm := &StorageManager{
		registered: make(map[string]time.Time),
		mongodb:    mongo,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/upload/", sm.uploadHandler)
	mux.HandleFunc("/download/", sm.downloadHandler)
	mux.HandleFunc("/register", sm.registerHandler)
	mux.HandleFunc("/usage", sm.usageHandler)

	sm.server = &http.Server{
		Addr:    ManagerAddr,
		Handler: mux,
	}
	return sm
}

// prettyNumber formats a number in a human-readable format.
func prettyNumber(size int) string {
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), unitChar[exp])
}

// AddStorage adds a new storage to the manager.
func (sm *StorageManager) AddStorage(url string, storage *Storage) error {
	sm.Lock()
	defer sm.Unlock()

	if _, found := sm.registered[url]; found {
		return errors.New("storage already registered")
	}

	sm.registered[url] = time.Now()
	sm.storages = append(sm.storages, storage)
	return nil
}

// registerHandler registers a new storage.
func (sm *StorageManager) registerHandler(w http.ResponseWriter, r *http.Request) {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)

	if r.Header.Get("Register") != "true" {
		http.Error(w, "Invalid Register header", http.StatusBadRequest)
		log.Printf("Invalid Register header: %v\n", r.Header.Get("Register"))
		return
	}

	limitStr, usedStr, addr := r.Header.Get("Limit"), r.Header.Get("Used"), r.Header.Get("Addr")
	if limitStr == "" || usedStr == "" || addr == "" {
		http.Error(w, "Missing required headers", http.StatusBadRequest)
		log.Println("Missing required headers")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		http.Error(w, "Invalid Limit header", http.StatusBadRequest)
		log.Printf("Invalid Limit header: %v\n", err)
		return
	}

	used, err := strconv.Atoi(usedStr)
	if err != nil {
		http.Error(w, "Invalid Used header", http.StatusBadRequest)
		log.Printf("Invalid Used header: %v\n", err)
		return
	}

	if net.ParseIP(ip).To4() == nil {
		ip = "[" + ip + "]"
	}

	url := "http://" + ip + addr
	storage := &Storage{
		URL:   url,
		Limit: limit,
		Used:  used,
	}

	if err = sm.AddStorage(url, storage); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		log.Printf("Failed to register storage: %v\n", err)
		return
	}
	log.Printf("Storage %s registered successfully\n", url)
}

// usageHandler returns the usage of the storages.
func (sm *StorageManager) usageHandler(w http.ResponseWriter, r *http.Request) {
	sm.RLock()
	defer sm.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "    ")
	encoder.Encode(sm.storages)
}

// uploadScheme distributes the file across the available storages according to their free space.
func (sm *StorageManager) uploadScheme(fileSize int) ([]*Scheme, error) {
	sm.RLock()
	defer sm.RUnlock()

	totalLimit, totalUsed := 0, 0
	for _, storage := range sm.storages {
		totalLimit += storage.Limit
		totalUsed += storage.Used
	}

	if fileSize > totalLimit - totalUsed {
		log.Printf("Not enough space to store file: %d > %d\n", fileSize, totalLimit-totalUsed)
		return nil, fmt.Errorf("Not enough space to store file: %d > %d\n", fileSize, totalLimit-totalUsed)
	}

	sortedStorages := make([]*Storage, len(sm.storages))
	copy(sortedStorages, sm.storages)
	sort.Slice(sortedStorages, func(i, j int) bool {
		return sortedStorages[i].Used < sortedStorages[j].Used
	})

	totalFree := 0
	for _, storage := range sortedStorages {
		storage.free = (storage.Limit - storage.Used) * 100 / storage.Limit
		totalFree += storage.free
		// log.Printf("Storage %s: Used: %d from %d free: %v%%\n", storage.URL, storage.Used, storage.Limit, storage.free)
	}

	numChunks := calculateNumChunks(fileSize, len(sortedStorages))
	remainingSize := fileSize
	var scheme []*Scheme

	for i := 0; i < numChunks; i++ {
		var size int
		storage := sortedStorages[i%len(sortedStorages)]

		if i == numChunks-1 {
			size = remainingSize
		} else {
			size = (fileSize * storage.free) / totalFree
			size = (size + chunkSize - 1) / chunkSize * chunkSize
			if size > remainingSize {
				size = remainingSize
			}
			remainingSize -= size
		}

		scheme = append(scheme, &Scheme{
			url:  storage.URL,
			size: size,
		})

		if remainingSize == 0 {
			break
		}
	}
	return scheme, nil
}

func calculateNumChunks(fileSize int, numStorages int) int {
	if fileSize < chunkSize {
		return 1
	}

	maxChunks := (fileSize + chunkSize - 1) / chunkSize
	if maxChunks > numStorages {
		// log.Printf("fileSize: %v numStorages: %v maxChunks: %d\n", fileSize, numStorages, numStorages)
		return numStorages
	}
	// log.Printf("fileSize: %v numStorages: %v maxChunks: %d\n", fileSize, numStorages, maxChunks)
	return numStorages
}

// Start starts the listener for the storage manager.
func (sm *StorageManager) Start() error {
	return sm.server.ListenAndServe()
}

// Store stores the file metadata in files and metadata collections.
func (sm *StorageManager) Store(filename string, fileInfo any) {
	var err error

	switch val := fileInfo.(type) {
	case string:
		fileInfo := &FileInfo{
			Hash: val,
			Name: filename,
		}
		err = sm.mongodb.InsertOne(fileInfo)

	case *FileInfo:
		for i := range val.Metadata {
			val.Metadata[i] = strings.Replace(val.Metadata[i], "upload", storedMark, 1)
		}
		err = sm.mongodb.InsertOne(val)
	}

	if err != nil {
		log.Printf("Failed to insert metadata into MongoDB: %v\n", err)
	}
}

// Load loads the file metadata.
func (sm *StorageManager) Load(filename string) (*FileInfo, error) {
	return sm.mongodb.Load(filename)
}

func (sm *StorageManager) Find(filename string, hash ...string) (*FileInfo, error) {
	if len(hash) > 0 {
		return sm.mongodb.FindOne(filename, hash[0])
	}
	return sm.mongodb.FindOne(filename)
}

func (sm *StorageManager) validateRequest(filename string, hash string) error {
	fileInfo, err := sm.Find(filename, hash)
	if err != nil {
		// log.Printf("DEBUG: Error loading file info for filename '%s' with hash '%s': %v", filename, hash, err)
		return err
	}

	if fileInfo.Name == "" {
		log.Printf("validateRequest: add file %s with hash %s", filename, hash)
		sm.Store(filename, fileInfo.Hash)
		return nil
	}
	return ErrAlreadyExist
}

// uploadHandler handles the file upload.
func (sm *StorageManager) uploadHandler(w http.ResponseWriter, r *http.Request) {
	var (
		scheme  []*Scheme
		err error
	)

	filename := filepath.Base(r.URL.Path)
	log.Printf("Received upload request for file: %s\n", filename)

	rollback := false
	defer func() {
		if rollback {
			go sm.rollbackScheme(scheme)
		}
	}()

	size := r.ContentLength
	if size <= 0 {
		http.Error(w, "Invalid Content-Length", http.StatusBadRequest)
		log.Printf("Invalid Content-Length: %v\n", size)
		return
	}

	hash := r.Header.Get("X-Hash")

	if err = sm.validateRequest(filename, hash); err == nil {
		return

	} else if err != mongo.ErrNoDocuments {
		log.Printf("validateRequest for %s: %v", filename, err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	scheme, err = sm.uploadScheme(int(size))
	if err != nil {
		http.Error(w, "Error uploading file", http.StatusInternalServerError)
		log.Printf("Error uploading file: %v\n", err)
		return
	}

	hasher := sha256.New()
	var metadata []string

	for _, target := range scheme {
		limitedReader := &io.LimitedReader{R: r.Body, N: int64(target.size)}

		storedHash, tmpfile, err := sm.storeChunk(w, target, hasher, limitedReader)
		if err != nil {
			log.Printf("Error storing chunk: %v", err)
			rollback = true
			return
		}

		target.tmpfile = tmpfile                          // temporary filename on the storage side
		target.url += "/" + storedMark + "/" + storedHash // url based on hash
		metadata = append(metadata, target.url)
	}

	for i, target := range scheme {
		log.Printf("Scheme[%d]: %v (%v)\n", i, target.url, target.size)
	}

	hash = hex.EncodeToString(hasher.Sum(nil))

	if _, err := sm.Find(filename, hash); err == nil {
		log.Printf("File with hash '%s' already exist. STORE & ROLLBACK", hash)
		sm.Store(filename, hash)
		rollback = true
		return
	}

	if err = sm.commitScheme(scheme); err != nil {
		http.Error(w, "Error committing chunks", http.StatusInternalServerError)
		log.Printf("Error committing chunks: %v\n", err)
		rollback = true
		return
	}

	fileInfo := &FileInfo{
		Hash:     hash,
		Name:     filename,
		Size:     int64(size),
		Metadata: metadata,
	}
	sm.Store(hash, fileInfo)

	log.Printf("filename: %s size: %v sha256: %v uploaded successfully\n", filename, size, hash)

	prettyJSON, _ := json.MarshalIndent(fileInfo, "", "    ")
	log.Printf(string(prettyJSON))
}

func (sm *StorageManager) storeChunk(w http.ResponseWriter, target *Scheme, hasher hash.Hash, body io.Reader) (storedHash, tmpfile string, err error) {
	segmentHasher := sha256.New()

	// Use MultiWriter to write to both hashers
	multiWriter := io.MultiWriter(segmentHasher, hasher)
	teeReader := io.TeeReader(body, multiWriter)

	resp, err := sm.storageRequest("PUT", target.url + "/upload/segment", teeReader, target.size)
	if err != nil {
		// log.Printf("Error storing chunk: %v", err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// log.Printf("Error storing chunk: %v", resp.StatusCode)
		return
	}

	storedHash = resp.Header.Get("X-Hash")

	if storedHash != hex.EncodeToString(segmentHasher.Sum(nil)) {
		// log.Printf("Error storing chunk: hash mismatch")
		return
	}
	return storedHash, resp.Header.Get("Filename"), nil
}

func (sm *StorageManager) updateStorage(target *Scheme, commit bool) {
	sm.Lock()
	defer sm.Unlock()

	for _, storage := range sm.storages {
		url, _ := url.Parse(target.url)
		if storage.URL == url.Host {
			if commit {
				storage.Used += target.size
			} else {
				storage.Used -= target.size
			}
			break
		}
	}
}

func (sm *StorageManager) commitScheme(scheme []*Scheme) error {
	for _, target := range scheme {
		url := strings.Replace(target.url, storedMark, "commit", 1)

		resp, err := sm.storageRequest("POST", url, nil, target.tmpfile)
		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to commit chunk, status code: %d", resp.StatusCode)
		}

		log.Printf("Committed chunk: %s\n", target.url)
		sm.updateStorage(target, true)
	}
	return nil
}

func (sm *StorageManager) rollbackScheme(scheme []*Scheme) {
	for _, target := range scheme {
		if target.tmpfile == "" {
			continue
		}
		url := strings.Replace(target.url, storedMark, "rollback", 1)

		resp, err := sm.storageRequest("DELETE", url, nil, target.tmpfile)
		if err != nil {
			log.Printf("rollbackScheme: %v", err)
			continue
		}
		resp.Body.Close()

		log.Printf("Rollback chunk: %v\n", target.url)
		sm.updateStorage(target, false)
	}
}

func (sm *StorageManager) storageRequest(method string, url string, body io.Reader, extra ...any) (*http.Response, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Distributed Storage System")

	if len(extra) > 0 {
		switch val := extra[0].(type) {
		case int:
			req.ContentLength = int64(val)

		case string:
			req.Header.Set("Filename", val)
		}
	}
	return client.Do(req)
}

func (sm *StorageManager) retrieveChunk(url string) (*http.Response, error) {
	resp, err := sm.storageRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download chunk, status code: %d", resp.StatusCode)
	}
	return resp, nil
}

// downloadHandler handles the file download.
func (sm *StorageManager) downloadHandler(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(r.URL.Path)

	fileInfo, err := sm.Load(filename)
	if err != nil {
		http.NotFound(w, r)
		log.Printf("File not found: %s\n", filename)
		return
	}

	w.Header().Add("Content-Length", strconv.FormatInt(fileInfo.Size, 10))
	w.Header().Add("X-Hash", fileInfo.Hash)
	w.Header().Add("Server", "Distributed Storage System")
	w.Header().Add("Content-Type", "application/octet-stream")

	var resp *http.Response
	for _, chunkURL := range fileInfo.Metadata {
		chunkURL = strings.Replace(chunkURL, storedMark, "download", 1)
		log.Printf("Retrieving chunk: %s", chunkURL)

		resp, err = sm.retrieveChunk(chunkURL)
		if err != nil {
			log.Printf("Error reading chunk %s: %v\n", chunkURL, err)
			return
		}

		hasher := sha256.New()
		multWriter := io.MultiWriter(w, hasher)
		if _, err := io.Copy(multWriter, resp.Body); err != nil && err != io.EOF {
			resp.Body.Close()
			log.Printf("Error writing chunk %s: %v\n", chunkURL, err)
			return
		}

		calculatedHash := hex.EncodeToString(hasher.Sum(nil))
		expectedHash := filepath.Base(chunkURL)
		if calculatedHash != expectedHash {
			resp.Body.Close()
			log.Printf("Hash mismatch for chunk %s\n", chunkURL)
			return
		}
		resp.Body.Close()
	}
	log.Printf("filename: %s size: %v sha256: %v downloaded successfully", filename, fileInfo.Size, fileInfo.Hash)
}

func main() {
	MongoURL = os.Getenv("MONGO_URL")
	if MongoURL == "" {
		log.Fatal("MONGO_URL environment variable not set")
	}

	ManagerAddr = os.Getenv("MANAGER_ADDR")
	if ManagerAddr == "" {
		log.Fatal("MANAGER_ADDR environment variable not set")
	}

	log.Printf("Starting storage manager on %v", ManagerAddr)
	manager := NewStorageManager()
	log.Fatal(manager.Start())
}

