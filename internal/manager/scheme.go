package manager

import (
	"errors"
)

// uploadScheme creates an uploading scheme for the given file size.
func (m *Manager) uploadScheme(fileSize int) (scheme []*Scheme, err error) {
    m.RLock()
    defer m.RUnlock()

    storagesCount := len(m.storages)

    if storagesCount == 0 {
        return nil, errors.New("no storages available")
    }

    scheme = make([]*Scheme, 0, storagesCount)
    storages := make([]*Storage, 0, storagesCount)

    var totalPercent float64
    var totalAvailableSpace int

    // Calculate available percent for each storage
    for _, storage := range m.storages {
		if storage.Used >= storage.Limit {
			continue
		}
        storage.availablePercent = 100 - (float64(storage.Used) / float64(storage.Limit) * 100)
        totalPercent += storage.availablePercent
        storages = append(storages, storage)

        totalAvailableSpace += storage.Limit - storage.Used
    }

    if totalAvailableSpace < fileSize {
        return nil, errors.New("not enough space available in storages")
    }

    // Calculate proportions for each storage
    assigned := 0
    for _, storage := range storages {
        storage.proportion = int(float64(fileSize) * (storage.availablePercent / totalPercent))
        storage.fractional = float64(fileSize) * (storage.availablePercent / totalPercent) - float64(storage.proportion)
        assigned += storage.proportion
    }

    // Distribute the remaining bytes to the storages
    remainder := fileSize - assigned
    for i := 0; i < remainder; i++ {
        storages[i % storagesCount].proportion++
    }

    for _, storage := range storages {
        if storage.proportion > 0 {
            scheme = append(scheme, &Scheme{
                URL:  storage.URL,
                Size: storage.proportion,
            })
			storage.Used += storage.proportion // if rollback, this will be reverted
        }
    }
    return scheme, nil
}
