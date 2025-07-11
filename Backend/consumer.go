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
	_, err := js.QueueSubscribe("jobs.created", "job-created-group", func(msg *nats.Msg) {
		var jobMsg JobMessage
		if err := json.Unmarshal(msg.Data, &jobMsg); err != nil {
			log.Printf("Failed to unmarshal job message: %v", err)
			msg.Nak()
			return
		}
		log.Printf("Received job: %+v", jobMsg.Data.Id)

		//Get the prompt for CV generation from database with field cvGenerationDefault = true
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
		cv, err := generateCVWithGemini(promptText)
		if err != nil {
			log.Printf("Error generating CV: %v", err)
			msg.Nak()
			return
		}
		log.Printf("Generated CV for Job : %v", jobMsg.Data.Id)

		//after generation, update the job with the generated CV
		_, err = db.Exec("UPDATE jobs SET cv = ?, cvGenerated = TRUE WHERE id = ?", cv, jobMsg.Data.Id)
		if err != nil {
			log.Printf("Failed to update job with generated CV: %v", err)
			msg.Nak()
			return
		}
		log.Printf("Job %d updated with generated CV", jobMsg.Data.Id)

		//after updating the job, send another message to jobs.cvgenerated to notify other services
		cvGeneratedMsg := JobMessage{
			Type: "cv_generated",
			Data: jobMsg.Data,
		}
		data, err := json.Marshal(cvGeneratedMsg)
		if err != nil {
			log.Printf("Failed to marshal CV generated message: %v", err)
			msg.Nak()
			return
		}
		if _, err := js.Publish("jobs.cvgenerated", data); err != nil {
			log.Printf("Failed to publish CV generated message: %v", err)
			msg.Nak()
			return
		}
		log.Printf("Published CV generated message for job %d", jobMsg.Data.Id)

		msg.Ack()
	}, nats.Durable("job-created-queue-consumer"), nats.ManualAck())
	if err != nil {
		return err
	}
	log.Println("Job created consumer started and listening on jobs.created")
	return nil
}

func startCvCreatedConsumer(js nats.JetStreamContext) error {
	_, err := js.QueueSubscribe("jobs.cvgenerated", "cv-created-group", func(msg *nats.Msg) {
		var jobMsg JobMessage
		if err := json.Unmarshal(msg.Data, &jobMsg); err != nil {
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

		score, err := generateCVWithGemini(scorePrompt)
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
	}, nats.Durable("cv-created-consumer"), nats.ManualAck())
	if err != nil {
		return err
	}
	log.Println("CV created consumer started and listening on jobs.cvgenerated")
	return nil
}
