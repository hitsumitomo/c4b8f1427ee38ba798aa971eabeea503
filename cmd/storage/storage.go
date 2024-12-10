package main

import (
	"dcloud/internal/storage"
	"log"
	"os"
)

func main() {
	addr := os.Getenv("STORAGE_ADDR")
	dir  := os.Getenv("STORAGE_DIR")
	url  := os.Getenv("REGISTER_URL")

	log.Printf("Storage %s Dir: %s RegisterURL: %s", addr, dir, url)

	if addr == "" || dir == "" || url == "" {
		log.Fatal("STORAGE_ADDR, STORAGE_DIR and REGISTER_URL environment variables must be set")
	}

	s, err := storage.New(addr, dir, url)
	if err != nil {
		log.Fatalf("Storage %s create error: %v", addr, err)
	}

	if err = s.Start(); err != nil {
		log.Fatalf("Storage %s start error: %v", addr, err)
	}
	select{}
}
