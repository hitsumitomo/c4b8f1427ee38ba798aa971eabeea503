package manager

import (
	"net/http"
	"sync"
	"time"

	"dcloud/internal/database"
)

type Manager struct {
	sync.RWMutex
	storages   map[string]*Storage

	server     *http.Server
	mongodb    *database.MongoDB
}

type Storage struct {
	Limit          int
	Used           int
	URL            string
	Registered     time.Time

	free             int
	availablePercent float64
	fractional       float64
	proportion       int
}

type Scheme struct {
    URL     string `json:"url"`
    Size    int    `json:"size"`
    Tmpfile string `json:"tmpfile"`
}
