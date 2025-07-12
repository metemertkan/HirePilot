package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var js jetstream.JetStream

func publishJobMessage(job Job) error {
	if js == nil {
		js = initJetStream()
	}

	// Create message payload
	message := map[string]interface{}{
		"type": "job_created",
		"data": job,
	}

	// Convert to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish to JetStream
	_, err = js.Publish(context.Background(), "jobs.created", payload)
	if err != nil {
		return err
	}

	log.Printf("Published job message for job ID: %d", job.Id)
	return nil
}

func initJetStream() jetstream.JetStream {
	// Get NATS URL from environment variable, default to localhost
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	// Connect to NATS
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Printf("Failed to connect to NATS: %v", err)
		return nil
	}

	js, err = jetstream.New(nc)
	if err != nil {
		log.Printf("Failed to create JetStream context: %v", err)
		return nil
	}

	stream, err := js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:      "JOBS",
		Subjects:  []string{"jobs.*"},
		Retention: jetstream.WorkQueuePolicy,
	})

	if err != nil {
		log.Printf("Failed to create stream: %v", err)
	} else {
		info, err := stream.Info(context.Background())
		if err != nil {
			log.Printf("Failed to get stream info: %v", err)
		}
		log.Printf("Created stream: %v", info.Config.Name)
	}
	return js
}
