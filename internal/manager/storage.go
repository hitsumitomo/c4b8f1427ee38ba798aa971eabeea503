package manager

import (
	"errors"
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

	for _, storage := range m.storages {
		url, _ := url.Parse(target.URL)
		if storage.URL == url.Host {
			if commited {
				storage.Used += target.Size
			} else {
				storage.Used -= target.Size
			}
			break
		}
	}
}
