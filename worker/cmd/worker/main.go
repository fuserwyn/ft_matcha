package main

import (
	"log"
	"os"
	"time"
)

func main() {
	log.Println("Worker starting...")

	// Placeholder: poll loop for sync jobs
	// Will be extended with: ES sync, Neo4j sync, email, fame rating
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Worker tick (placeholder)")
		_ = os.Getenv("DATABASE_URL") // use env to avoid unused warning
	}
}
