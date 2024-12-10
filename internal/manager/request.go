package manager

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"log"
	"net/http"
	"strings"
)

// storageRequest sends an HTTP request with the specified method, URL, and body.
// It also sets additional headers if provided.
func (m *Manager) storageRequest(method string, url string, body io.Reader, extra ...any) (*http.Response, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "DCLOUD")

	if len(extra) > 0 {
		switch val := extra[0].(type) {
		case int:
			req.ContentLength = int64(val)

		case string:
			req.Header.Set("X-Filename", val)
		}
	}
	return client.Do(req)
}

// commitScheme commits a scheme by sending a POST request to the commit URL.
func (m *Manager) commitScheme(scheme []*Scheme) error {
	for _, target := range scheme {
		url := strings.Replace(target.URL, storedMark, "commit", 1)

		resp, err := m.storageRequest(http.MethodPost, url, nil, target.Tmpfile)
		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to commit chunk, status code: %d", resp.StatusCode)
		}

		log.Printf("Committed chunk: %s\n", target.URL)
		m.updateStorage(target, true)
	}
	return nil
}

// rollbackScheme rolls back a schemes by sending a DELETE request to the rollback URL.
func (m *Manager) rollbackScheme(scheme []*Scheme) {
	for _, target := range scheme {
		if target.Tmpfile == "" {
			continue
		}
		url := strings.Replace(target.URL, storedMark, "rollback", 1)

		resp, err := m.storageRequest(http.MethodDelete, url, nil, target.Tmpfile)
		if err != nil {
			log.Printf("rollbackScheme: %v", err)
			continue
		}
		resp.Body.Close()

		log.Printf("Rollback chunk: %v\n", target.URL)
		m.updateStorage(target, false)
	}
}

// retrieveChunk retrieves a chunk from the specified URL
func (m *Manager) retrieveChunk(url string) (*http.Response, error) {
	resp, err := m.storageRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download chunk, status code: %d", resp.StatusCode)
	}
	return resp, nil
}

// storeChunk stores a chunk by sending a PUT request with the chunk data.
// It also calculates and verifies the hash of the stored chunk.
func (m *Manager) storeChunk(_ http.ResponseWriter, target *Scheme, hasher hash.Hash, body io.Reader) (storedHash, tmpFilePath string, err error) {
	segmentHasher := sha256.New()

	// Use MultiWriter to write to both hashers
	multiWriter := io.MultiWriter(segmentHasher, hasher)
	teeReader := io.TeeReader(body, multiWriter)

	resp, err := m.storageRequest(http.MethodPut, target.URL + "/segment", teeReader, target.Size)
	if err != nil {
		return "", "", err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to store chunk, status: %s", resp.Status)
	}

	storedHash = resp.Header.Get("X-Hash")

	if storedHash != hex.EncodeToString(segmentHasher.Sum(nil)) {
		return "", "", errors.New("hash mismatch")
	}
	return storedHash, resp.Header.Get("X-Filename"), nil
}
