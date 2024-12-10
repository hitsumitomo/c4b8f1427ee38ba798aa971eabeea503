package manager

import (
	"dcloud/internal/file"
	"log"
	"strings"
)

// Store stores the file info in files and metadata collections.
func (m *Manager) Store(filename string, fileInfo any) {
	var err error

	switch val := fileInfo.(type) {
	case string:
		fileInfo := &file.Info{
			Hash: val,
			Name: filename,
		}
		err = m.mongodb.Store(fileInfo)

	case *file.Info:
		for i := range val.Metadata {
			val.Metadata[i] = strings.Replace(val.Metadata[i], "upload", storedMark, 1)
		}
		err = m.mongodb.Store(val)
	}

	if err != nil {
		log.Printf("Failed to insert metadata into MongoDB: %v\n", err)
	}
}

// Find finds the file info from the MongoDB.
func (m *Manager) Load(filename string, hash ...string) (*file.Info, error) {
	return m.mongodb.Load(filename, hash...)
}
