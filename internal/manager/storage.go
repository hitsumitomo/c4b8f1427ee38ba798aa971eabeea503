package manager

import (
	"errors"
	"log"
	"net/url"
	"time"
)

// addStorage adds a new storage to the manager.
func (m *Manager) addStorage(url string, storage *Storage) error {
	m.Lock()
	defer m.Unlock()

	if _, found := m.storages[url]; found {
		return errors.New("storage already registered")
	}

	storage.Registered = time.Now()
	m.storages[url] = storage
	return nil
}

// updateStorage updates the storage usage.
func (m *Manager) updateStorage(target *Scheme, commited bool) {
	m.Lock()
	defer m.Unlock()

	if !commited {
		u, err := url.Parse(target.URL)
		if err != nil {
			log.Printf("updateStorage: Failed to parse URL: %v", err)
			return
		}
		baseURL := u.Scheme + "://" + u.Host
		if storage, found := m.storages[baseURL]; found {
			storage.Used -= target.Size
		}
	}
}
