package main

import (
	"log"
)

func main() {
	js := initJetStream()
	initDB()

	if _, err := startCvCreatedConsumer(js); err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}

	// Block forever
	select {}
}
