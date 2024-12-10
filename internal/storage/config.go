package storage

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// New creates a new Storage instance, initializes it, and sets up HTTP handlers.
func New(addr, dir, url string) (s *Storage, err error) {
	s = &Storage{
		Limit: 10 * 1024 * 1024 * 1024, // 10 GB
		Addr:  addr,
		Dir:   dir,
		RegisterURL: url,
	}

	if err = s.initStorage(); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.routeHandler)
	mux.HandleFunc("/rollback/", s.rollbackHandler)
	mux.HandleFunc("/commit/", s.commitHandler)

	s.server = &http.Server{
		Addr:    s.Addr,
		Handler: mux,
	}
	return s, nil
}

// initStorage initializes the storage directory and calculates the used space.
func (s *Storage) initStorage() (err error){
	var total int64

	if err = os.MkdirAll(s.Dir, 0755); err != nil {
		return err
	}

	err = filepath.Walk(s.Dir, func(path string, info os.FileInfo, err error) error {
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
		return err
	}

	s.Used = total

	if s.Used > s.Limit {
		s.Used = s.Limit
	}
	return nil
}

// register sends a registration request to the specified URL with storage details.
func (s *Storage) register() error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest(http.MethodConnect, s.RegisterURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Register", "true")
	req.Header.Set("X-Addr", s.Addr)
	req.Header.Set("X-Limit", strconv.FormatInt(s.Limit, 10))
	req.Header.Set("X-Used", strconv.FormatInt(s.Used, 10))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("register failed with status %v", resp.Status)
	}
	return nil
}

// Start begins the HTTP server and registers the storage.
func (s *Storage) Start() (err error) {
	go func () {
		err = s.server.ListenAndServe()
	}()

	time.Sleep(500 * time.Millisecond)

	if err != nil {
		return err
	}

	if err = s.register(); err != nil {
		return err
	}
	log.Printf("Storage %s successfully registered", s.Addr)
	return nil
}