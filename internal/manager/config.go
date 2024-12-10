package manager

import (
	"log"
	"net/http"
	"time"

	"dcloud/internal/database"
)

const (
	timeout    = 10 * time.Second
	storedMark = "[STORED]"
)

// New creates a new storage manager.
func New(addr, mongodb string) (m *Manager, err error) {
	m = &Manager{
		storages: make(map[string]*Storage),
	}

	m.mongodb, err = database.Connect(mongodb)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", m.routeHandler)
	mux.HandleFunc("/register", m.storageRegister)
	mux.HandleFunc("/usage", m.storageUsage)

	m.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return m, nil
}

// Start starts http server.
func (m *Manager) Start() {
	log.Printf("Manager listening on %s\n", m.server.Addr)
	m.server.ListenAndServe()
}

