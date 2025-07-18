package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type JobMessage struct {
	Type string `json:"type"`
	Data Job    `json:"data"`
}

func startJobCreatedConsumer(js jetstream.JetStream) (jetstream.ConsumeContext, error) {

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "job-created-consumer",
		Durable:        "job-created-consumer",
		FilterSubjects: []string{"jobs.created"},
		AckWait:        5 * time.Minute,
	})

	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		var jobMsg JobMessage
		if err := json.Unmarshal(msg.Data(), &jobMsg); err != nil {
			log.Printf("Failed to unmarshal job message: %v", err)
			msg.Nak()
			return
		}
		log.Printf("Received job: %+v", jobMsg.Data.Id)
		// Get the prompt for CV generation from database with field cvGenerationDefault = true
		var prompt Prompt
		err := db.QueryRow("SELECT prompt FROM prompts WHERE cvGenerationDefault = true LIMIT 1").Scan(&prompt.Prompt)
		if err != nil {
			log.Printf("Failed to get prompt for CV generation: %v", err)
			msg.Nak()
			return
		}

		// Example: Use generateCVWithGemini
		promptText := prompt.Prompt + "\n\n" +
			"Title: " + jobMsg.Data.Title + "\n" +
			"Company: " + jobMsg.Data.Company + "\n" +
			"Description: " + jobMsg.Data.Description + "\n"
		cv, err := generateWithGemini(promptText)
		if err != nil {
			log.Printf("Error generating CV: %v", err)
			msg.Nak()
			return
		}

		log.Printf("Generated CV for Job : %v", jobMsg.Data.Id)

		// after generation, update the job with the generated CV
		_, err = db.Exec("UPDATE jobs SET cv = ?, cvGenerated = TRUE WHERE id = ?", cv, jobMsg.Data.Id)
		if err != nil {
			log.Printf("Failed to update job with generated CV: %v", err)
			msg.Nak()
			return
		}
		log.Printf("Job %d updated with generated CV", jobMsg.Data.Id)

		// Check if the feature for score generation is enabled
		// and publish a message to jobs.cvgenerated if it is
		var feature Feature
		err = db.QueryRow("SELECT value FROM features WHERE name = 'scoreGeneration'").
			Scan(&feature.Value)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("DB query error: %v", err)
			return
		}

		if feature.Value {
			result := generateCvGeneratedMessage(jobMsg, js)
			if !result {
				log.Printf("Failed to publish CV generated message for job %d", jobMsg.Data.Id)
			}
			log.Printf("Published CV generated message for job %d", jobMsg.Data.Id)
		}

		msg.Ack()
	})

}

func generateCvGeneratedMessage(jobMsg JobMessage, js jetstream.JetStream) bool {
	// after updating the job, send another message to jobs.cvgenerated to notify other services
	cvGeneratedMsg := JobMessage{
		Type: "cv_generated",
		Data: jobMsg.Data,
	}
	data, err := json.Marshal(cvGeneratedMsg)
	if err != nil {
		log.Printf("Failed to marshal CV generated message: %v", err)
		return false
	}
	if _, err := js.Publish(context.Background(), "jobs.cvgenerated", data); err != nil {
		log.Printf("Failed to publish CV generated message: %v", err)
		return false
	}
	log.Printf("Published CV generated message for job %d", jobMsg.Data.Id)
	return true
}

func startCvCreatedConsumer(js jetstream.JetStream) (jetstream.ConsumeContext, error) {

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "cv-created-consumer",
		Durable:        "cv-created-consumer",
		FilterSubjects: []string{"jobs.cvgenerated"},
		AckWait:        5 * time.Minute,
	})

	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		var jobMsg JobMessage
		if err := json.Unmarshal(msg.Data(), &jobMsg); err != nil {
			log.Printf("Failed to unmarshal job message: %v", err)
			msg.Nak()
			return
		}
		log.Printf("Received job: %+v", jobMsg.Data.Id)

		//Get the prompt for Score generation from database with field cvGenerationDefault = true
		var prompt Prompt
		err := db.QueryRow("SELECT prompt FROM prompts WHERE scoreGenerationDefault = true LIMIT 1").Scan(&prompt.Prompt)
		if err != nil {
			log.Printf("Failed to get prompt for Score generation: %v", err)
			msg.Nak()
			return
		}

		scorePrompt := prompt.Prompt + "\n\n" +
			"Job Description: " + jobMsg.Data.Description + "\n" +
			"CV: " + jobMsg.Data.Cv + "\n"

		score, err := generateWithGemini(scorePrompt)
		if err != nil {
			log.Printf("Gemini API error: %v", err)
			return
		}
		log.Printf("Generated Score : %s", score)

		_, err = db.Exec("UPDATE jobs SET score = ? WHERE id = ?", score, jobMsg.Data.Id)
		if err != nil {
			log.Printf("Db update error: %v", err)
			return
		}
		log.Printf("Job %d updated with generated Score", jobMsg.Data.Id)
		msg.Ack()
	})
}
