package manager

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
)

// registerHandler registers a new storage.
func (m *Manager) storageRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)

	register := r.Header.Get("X-Register")
	limitStr := r.Header.Get("X-Limit")
	usedStr  := r.Header.Get("X-Used")
	addr     := r.Header.Get("X-Addr")

	if register != "true" {
		log.Printf("Invalid Register header: %v", register)
		http.Error(w, "Invalid Register header", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		log.Printf("Invalid Limit header: %v", err)
		http.Error(w, "Invalid Limit header", http.StatusBadRequest)
		return
	}

	used, err := strconv.Atoi(usedStr)
	if err != nil {
		log.Printf("Invalid Used header: %v", err)
		http.Error(w, "Invalid Used header", http.StatusBadRequest)
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

	if err = m.addStorage(url, storage); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		log.Printf("Failed to register storage: %v", err)
		return
	}
	log.Printf("Manager successfully registered the storage %s", url)
}

// usageHandler returns the usage of the storages.
func (sm *Manager) storageUsage(w http.ResponseWriter, r *http.Request) {
	sm.RLock()
	defer sm.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "    ")
	encoder.Encode(sm.storages)
}
