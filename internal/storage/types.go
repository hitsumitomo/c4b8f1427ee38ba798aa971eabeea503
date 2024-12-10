package storage

import (
	"net/http"
	"time"
)

type Storage struct {
	Limit       int64
	Used        int64
	Addr        string
	Dir         string
	RegisterURL string
	Registered  time.Time

	server *http.Server
}

