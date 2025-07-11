package main

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

type JobMessage struct {
	Type string `json:"type"`
	Data Job    `json:"data"`
}

func startJobCreatedConsumer(js nats.JetStreamContext) error {
	_, err := js.Subscribe("jobs.created", func(msg *nats.Msg) {
		var jobMsg JobMessage
		if err := json.Unmarshal(msg.Data, &jobMsg); err != nil {
			log.Printf("Failed to unmarshal job message: %v", err)
			msg.Nak()
			return
		}
		log.Printf("Received job: %+v", jobMsg.Data)

		// Example: Use generateCVWithGemini
		prompt := "Generate a professional CV for the following job:\n" +
			"Title: " + jobMsg.Data.Title + "\n" +
			"Company: " + jobMsg.Data.Company + "\n" +
			"Description: " + jobMsg.Data.Description + "\n"
		cv, err := generateCVWithGemini(prompt)
		if err != nil {
			log.Printf("Error generating CV: %v", err)
			msg.Nak()
			return
		}
		log.Printf("Generated CV: %s", cv)
		msg.Ack()
	}, nats.Durable("job-created-consumer"), nats.ManualAck())
	if err != nil {
		return err
	}
	log.Println("Job created consumer started and listening on jobs.created")
	return nil
}
