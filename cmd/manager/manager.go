package main

import (
	"log"
	"os"

	"dcloud/internal/manager"
)

func main() {
	addr := os.Getenv("MANAGER_ADDR")
	mondodb := os.Getenv("MONGO_URL")
	if addr == "" || mondodb == "" {
		log.Fatal("MANAGER_ADDR and MONGO_URL environment variables must be set")
	}

	m, err := manager.New(addr, mondodb)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}
	m.Start()
}
