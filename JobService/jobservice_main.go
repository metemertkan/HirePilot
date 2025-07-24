package main

import (
	"encoding/json"
	"fmt"
	"log"

	sharedDB "github.com/hirepilot/shared/db"
	sharedNats "github.com/hirepilot/shared/nats"
	"github.com/hirepilot/shared/models"
)

type JobStatusUpdateMessage struct {
	Type string `json:"type"`
	Data sharedNats.JobStatusUpdateRequest `json:"data"`
}

func main() {
	log.Println("Starting Job Service...")

	// Initialize shared database
	sharedDB.InitDB()

	// Initialize shared NATS JetStream
	sharedNats.InitJetStream()
	defer sharedNats.Close()

	// Subscribe to job creation requests
	_, err := sharedNats.SubscribeToJobCreationRequestsGeneric(func(data []byte) error {
		log.Printf("Received job creation request")

		// Parse the message
		var message map[string]interface{}
		if err := json.Unmarshal(data, &message); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			return err
		}

		// Extract job data
		jobData, ok := message["data"].(map[string]interface{})
		if !ok {
			log.Printf("Invalid job data format")
			return fmt.Errorf("invalid job data format")
		}

		// Process the job creation request
		if err := handleJobCreation(jobData); err != nil {
			log.Printf("Error handling job creation: %v", err)
			return err
		}

		log.Printf("Job creation handled successfully")
		return nil
	})

	if err != nil {
		log.Fatalf("Failed to subscribe to job creation messages: %v", err)
	}

	// Subscribe to job status update requests
	_, err = sharedNats.SubscribeToJobStatusUpdateRequestsGeneric(func(data []byte) error {
		log.Printf("Received job status update request")

		var statusUpdateMsg JobStatusUpdateMessage
		if err := json.Unmarshal(data, &statusUpdateMsg); err != nil {
			log.Printf("Error unmarshaling job status update message: %v", err)
			return err
		}

		// Process the job status update request
		if err := handleJobStatusUpdate(statusUpdateMsg.Data); err != nil {
			log.Printf("Error handling job status update: %v", err)
			return err
		}

		log.Printf("Job status update handled successfully")
		return nil
	})

	if err != nil {
		log.Fatalf("Failed to subscribe to job status update messages: %v", err)
	}

	log.Println("Job Service subscribed to job creation and status update messages")
	log.Println("Job Service is running. Press Ctrl+C to exit.")

	// Keep the service running
	select {}
}

func handleJobCreation(jobData map[string]interface{}) error {
	// Extract job fields
	title, _ := jobData["title"].(string)
	company, _ := jobData["company"].(string)
	link, _ := jobData["link"].(string)
	description, _ := jobData["description"].(string)

	log.Printf("Processing job creation: %s at %s", title, company)

	// Insert job into database using shared library
	id, err := sharedDB.InsertJob(title, company, link, description)
	if err != nil {
		return err
	}

	log.Printf("Job saved to database with ID: %d", id)

	// Create complete job object for further processing
	job := models.Job{
		Id:          int(id),
		Title:       title,
		Company:     company,
		Link:        link,
		Status:      models.JobStatusOpen,
		CvGenerated: false,
		Cv:          "",
		Description: description,
	}

	// Check if CV generation feature is enabled using shared library
	cvGenerationEnabled, err := sharedDB.GetFeatureValue("cvGeneration")
	if err != nil {
		log.Printf("Warning: Could not check CV generation feature: %v", err)
		return nil // Don't fail the whole process for this
	}

	// Publish job created event for CV generation if enabled
	if cvGenerationEnabled {
		if err := sharedNats.PublishJobMessage(job); err != nil {
			log.Printf("Warning: Failed to publish job message for CV generation: %v", err)
		} else {
			log.Printf("Job message published for CV generation")
		}
	}

	return nil
}

func handleJobStatusUpdate(statusUpdate sharedNats.JobStatusUpdateRequest) error {
	log.Printf("Processing job status update: Job ID %d to status %s", statusUpdate.JobID, statusUpdate.Status)

	// Update job status in database using shared library
	err := sharedDB.UpdateJobStatus(statusUpdate.JobID, statusUpdate.Status)
	if err != nil {
		return err
	}

	log.Printf("Job %d status updated to %s in database", statusUpdate.JobID, statusUpdate.Status)
	return nil
}