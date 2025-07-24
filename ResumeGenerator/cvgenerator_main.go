package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	sharedAI "github.com/hirepilot/shared/ai"
	sharedDB "github.com/hirepilot/shared/db"
	"github.com/hirepilot/shared/models"
	sharedNats "github.com/hirepilot/shared/nats"
)

type JobMessage struct {
	Type string     `json:"type"`
	Data models.Job `json:"data"`
}

type CVGenerationRequestMessage struct {
	Type string                         `json:"type"`
	Data sharedNats.CVGenerationRequest `json:"data"`
}

func main() {
	// Initialize shared database
	sharedDB.InitDB()

	// Initialize shared NATS JetStream
	sharedNats.InitJetStream()

	if _, err := sharedNats.SubscribeToJobsCreatedForCVGeneric(handleJobCreated); err != nil {
		log.Fatalf("Failed to start job created consumer: %v", err)
	}

	if _, err := sharedNats.SubscribeToCVGenerationRequestsGeneric(handleCVGenerationRequest); err != nil {
		log.Fatalf("Failed to start CV generation request consumer: %v", err)
	}

	log.Println("ResumeGenerator started successfully")
	// Block forever
	select {}
}

func handleJobCreated(data []byte) error {
	var jobMsg JobMessage
	if err := json.Unmarshal(data, &jobMsg); err != nil {
		log.Printf("Failed to unmarshal job message: %v", err)
		return err
	}
	log.Printf("Received job: %+v", jobMsg.Data.Id)

	// Get the default CV generation prompt from database
	cvPrompt, err := sharedDB.GetDefaultCVPrompt()
	if err == sharedDB.ErrNotFound {
		log.Printf("No default CV generation prompt found")
		return fmt.Errorf("no default CV generation prompt found")
	} else if err != nil {
		log.Printf("Failed to get default CV prompt: %v", err)
		return err
	}

	// Use shared AI client to generate CV
	promptText := cvPrompt.Prompt + "\n\n" +
		"Title: " + jobMsg.Data.Title + "\n" +
		"Company: " + jobMsg.Data.Company + "\n" +
		"Description: " + jobMsg.Data.Description + "\n"

	aiClient, err := sharedAI.DefaultClient()
	if err != nil {
		log.Printf("AI client error: %v", err)
		return err
	}

	cv, err := aiClient.Generate(promptText)
	if err != nil {
		log.Printf("Error generating CV: %v", err)
		return err
	}

	log.Printf("Generated CV for Job : %v", jobMsg.Data.Id)

	// after generation, update the job with the generated CV using shared DB
	err = sharedDB.UpdateJobCV(jobMsg.Data.Id, cv)
	if err != nil {
		log.Printf("Failed to update job with generated CV: %v", err)
		return err
	}
	log.Printf("Job %d updated with generated CV", jobMsg.Data.Id)

	// Always publish CV generated message for PDF generation and other services
	// Update job data with generated CV and publish message
	jobMsg.Data.Cv = cv
	jobMsg.Data.CvGenerated = true

	err = sharedNats.PublishCVGeneratedMessage(models.Job{
		Id:          jobMsg.Data.Id,
		Title:       jobMsg.Data.Title,
		Company:     jobMsg.Data.Company,
		Link:        jobMsg.Data.Link,
		Status:      jobMsg.Data.Status,
		CvGenerated: jobMsg.Data.CvGenerated,
		Cv:          jobMsg.Data.Cv,
		Description: jobMsg.Data.Description,
		CreatedAt:   jobMsg.Data.CreatedAt,
		AppliedAt:   jobMsg.Data.AppliedAt,
	})
	if err != nil {
		log.Printf("Failed to publish CV generated message for job %d: %v", jobMsg.Data.Id, err)
		// Don't return error here - CV generation was successful, just message publishing failed
	} else {
		log.Printf("Published CV generated message for job %d", jobMsg.Data.Id)
	}

	return nil
}

func handleCVGenerationRequest(data []byte) error {
	var cvReqMsg CVGenerationRequestMessage
	if err := json.Unmarshal(data, &cvReqMsg); err != nil {
		log.Printf("Failed to unmarshal CV generation request message: %v", err)
		return err
	}
	log.Printf("Received CV generation request for job ID: %s", cvReqMsg.Data.JobID)

	// Fetch job from database using shared DB
	jobID, err := strconv.Atoi(cvReqMsg.Data.JobID)
	if err != nil {
		log.Printf("Invalid job ID: %s", cvReqMsg.Data.JobID)
		return err
	}

	job, err := sharedDB.GetJobByID(jobID)
	if err == sharedDB.ErrNotFound {
		log.Printf("Job not found for ID: %s", cvReqMsg.Data.JobID)
		return err
	} else if err != nil {
		log.Printf("DB query error: %v", err)
		return err
	}

	var promptText string
	if cvReqMsg.Data.PromptID != nil {
		// Fetch specific prompt
		prompt, err := sharedDB.GetPromptByID(*cvReqMsg.Data.PromptID)
		if err == sharedDB.ErrNotFound {
			log.Printf("Prompt not found for ID %d", *cvReqMsg.Data.PromptID)
			return err
		} else if err != nil {
			log.Printf("Failed to get prompt by ID %d: %v", *cvReqMsg.Data.PromptID, err)
			return err
		}
		promptText = prompt.Prompt
	} else {
		// Get default CV generation prompt
		cvPrompt, err := sharedDB.GetDefaultCVPrompt()
		if err == sharedDB.ErrNotFound {
			log.Printf("No default CV generation prompt found")
			promptText = "Generate a professional CV for the following job:"
		} else if err != nil {
			log.Printf("Failed to get default CV prompt: %v", err)
			return err
		} else {
			promptText = cvPrompt.Prompt
		}
	}

	// Generate CV using AI
	fullPrompt := promptText + "\n\n" +
		"Title: " + job.Title + "\n" +
		"Company: " + job.Company + "\n" +
		"Description: " + job.Description + "\n"

	aiClient, err := sharedAI.DefaultClient()
	if err != nil {
		log.Printf("AI client error: %v", err)
		return err
	}

	cv, err := aiClient.Generate(fullPrompt)
	if err != nil {
		log.Printf("Error generating CV: %v", err)
		return err
	}

	log.Printf("Generated CV for Job ID: %s", cvReqMsg.Data.JobID)

	// Update job with generated CV using shared DB
	jobID, err = strconv.Atoi(cvReqMsg.Data.JobID)
	if err != nil {
		log.Printf("Invalid job ID: %s", cvReqMsg.Data.JobID)
		return err
	}

	err = sharedDB.UpdateJobCV(jobID, cv)
	if err != nil {
		log.Printf("Failed to update job with generated CV: %v", err)
		return err
	}
	log.Printf("Job %s updated with generated CV", cvReqMsg.Data.JobID)

	// Publish CV generated message using shared NATS
	job.Cv = cv
	job.CvGenerated = true

	err = sharedNats.PublishCVGeneratedMessage(models.Job{
		Id:          job.Id,
		Title:       job.Title,
		Company:     job.Company,
		Link:        job.Link,
		Status:      job.Status,
		CvGenerated: job.CvGenerated,
		Cv:          job.Cv,
		Description: job.Description,
		CreatedAt:   job.CreatedAt,
		AppliedAt:   job.AppliedAt,
	})
	if err != nil {
		log.Printf("Failed to publish CV generated message for job %s: %v", cvReqMsg.Data.JobID, err)
		// Don't return error here - CV generation was successful, just message publishing failed
	} else {
		log.Printf("Published CV generated message for job %s", cvReqMsg.Data.JobID)
	}

	return nil
}
