package manager

import (
	"crypto/sha256"
	"dcloud/internal/file"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

var ErrAlreadyExist = errors.New("File already exists")

// routeHandler handles the incoming requests and routes them to the appropriate handler.
func (m *Manager) routeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.downloadHandler(w, r)

	case http.MethodPut:
		m.uploadHandler(w, r)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// uploadHandler handles the file upload.
func (m *Manager) uploadHandler(w http.ResponseWriter, r *http.Request) {
	var (
		scheme  []*Scheme
		err error
	)

	filename := filepath.Base(r.URL.Path)
	log.Printf("Received upload request for file: %s", filename)

	rollback := false
	defer func() {
		if rollback {
			go m.rollbackScheme(scheme)
			log.Printf("Rollback scheme for %s", filename)
			http.Error(w, "Error uploading file", http.StatusInternalServerError)
		}
	}()

	size := r.ContentLength
	if size <= 0 {
		log.Printf("Invalid Content-Length: %v", size)
		http.Error(w, "Invalid Content-Length", http.StatusBadRequest)
		return
	}

	hash := r.Header.Get("X-Hash")

	if err = m.validateRequest(filename, hash); err == nil {
		return

	} else if err != mongo.ErrNoDocuments {
		log.Printf("validateRequest for %s: %v", filename, err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	scheme, err = m.uploadScheme(int(size))
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInsufficientStorage)
		return
	}

	hasher := sha256.New()
	var metadata []string

	for _, target := range scheme {
		limitedReader := &io.LimitedReader{R: r.Body, N: int64(target.Size)}

		storedHash, tmpFilePath, err := m.storeChunk(w, target, hasher, limitedReader)
		if err != nil {
			log.Printf("Error storing chunk: %v", err)
			rollback = true
			return
		}

		target.Tmpfile = tmpFilePath                      // temporary filename on the storage side
		target.URL += "/" + storedMark + "/" + storedHash // url based on hash
		metadata = append(metadata, target.URL)
	}

	for i, target := range scheme {
		log.Printf("Scheme[%d]: %v (%v)", i, target.URL, target.Size)
	}

	hash = hex.EncodeToString(hasher.Sum(nil))

	if _, err := m.Load(filename, hash); err == nil {
		log.Printf("File with hash '%s' already exist. STORE & ROLLBACK", hash)
		m.Store(filename, hash)
		rollback = true
		return
	}

	if err = m.commitScheme(scheme); err != nil {
		http.Error(w, "Error committing chunks", http.StatusInternalServerError)
		log.Printf("Error committing chunks: %v", err)
		rollback = true
		return
	}

	fileInfo := &file.Info{
		Hash:     hash,
		Name:     filename,
		Size:     int64(size),
		Metadata: metadata,
	}
	m.Store(hash, fileInfo)

	log.Printf("filename: %s size: %v sha256: %v uploaded successfully", filename, size, hash)

	prettyJSON, _ := json.MarshalIndent(fileInfo, "", "    ")
	log.Print(string(prettyJSON))
}

// downloadHandler handles the file download.
func (m *Manager) downloadHandler(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(r.URL.Path)

	fileInfo, err := m.Load(filename)
	if err != nil {
		http.NotFound(w, r)
		log.Printf("File not found: %s", filename)
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

		resp, err = m.retrieveChunk(chunkURL)
		if err != nil {
			log.Printf("Error reading chunk %s: %v", chunkURL, err)
			return
		}

		hasher := sha256.New()
		multWriter := io.MultiWriter(w, hasher)
		if _, err := io.Copy(multWriter, resp.Body); err != nil && err != io.EOF {
			resp.Body.Close()
			log.Printf("Error writing chunk %s: %v", chunkURL, err)
			return
		}

		calculatedHash := hex.EncodeToString(hasher.Sum(nil))
		expectedHash := filepath.Base(chunkURL)
		if calculatedHash != expectedHash {
			resp.Body.Close()
			log.Printf("Hash mismatch for chunk %s", chunkURL)
			return
		}
		resp.Body.Close()
	}
	log.Printf("filename: %s size: %v sha256: %v downloaded successfully", filename, fileInfo.Size, fileInfo.Hash)
}

// validateRequest checks if the file already exists in the database.
func (m *Manager) validateRequest(filename string, hash string) error {
	fileInfo, err := m.Load(filename, hash)
	if err != nil {
		return err
	}

	if fileInfo.Name != "" {
		return ErrAlreadyExist
	}

	log.Printf("validateRequest: add file %s with hash %s", filename, hash)
	m.Store(filename, fileInfo.Hash)
	return nil
}
