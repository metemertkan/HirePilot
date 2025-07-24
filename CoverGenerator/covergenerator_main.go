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

type CoverGenerationRequestMessage struct {
	Type string                            `json:"type"`
	Data sharedNats.CoverGenerationRequest `json:"data"`
}

func main() {
	// Initialize shared database
	sharedDB.InitDB()

	// Initialize shared NATS JetStream
	sharedNats.InitJetStream()

	if _, err := sharedNats.SubscribeToJobsCreatedForCoverGeneric(handleJobCreated); err != nil {
		log.Fatalf("Failed to start job created consumer: %v", err)
	}

	if _, err := sharedNats.SubscribeToCoverGenerationRequestsGeneric(handleCoverGenerationRequest); err != nil {
		log.Fatalf("Failed to start cover generation request consumer: %v", err)
	}

	log.Println("CoverGenerator started successfully")
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

	// Get the default cover letter generation prompt from database
	coverPrompt, err := sharedDB.GetDefaultCoverPrompt()
	if err == sharedDB.ErrNotFound {
		log.Printf("No default cover letter generation prompt found")
		return fmt.Errorf("no default cover letter generation prompt found")
	} else if err != nil {
		log.Printf("Failed to get default cover prompt: %v", err)
		return err
	}

	// Use shared AI client to generate cover letter
	promptText := coverPrompt.Prompt + "\n\n" +
		"Title: " + jobMsg.Data.Title + "\n" +
		"Company: " + jobMsg.Data.Company + "\n" +
		"Description: " + jobMsg.Data.Description + "\n"

	aiClient, err := sharedAI.DefaultClient()
	if err != nil {
		log.Printf("AI client error: %v", err)
		return err
	}

	coverLetter, err := aiClient.Generate(promptText)
	if err != nil {
		log.Printf("Error generating cover letter: %v", err)
		return err
	}

	log.Printf("Generated cover letter for Job : %v", jobMsg.Data.Id)

	// after generation, update the job with the generated cover letter using shared DB
	err = sharedDB.UpdateJobCoverLetter(jobMsg.Data.Id, coverLetter)
	if err != nil {
		log.Printf("Failed to update job with generated cover letter: %v", err)
		return err
	}
	log.Printf("Job %d updated with generated cover letter", jobMsg.Data.Id)

	// Update job data with generated cover letter and publish message
	jobMsg.Data.CoverLetter = coverLetter

	err = sharedNats.PublishCoverGeneratedMessage(models.Job{
		Id:          jobMsg.Data.Id,
		Title:       jobMsg.Data.Title,
		Company:     jobMsg.Data.Company,
		Link:        jobMsg.Data.Link,
		Status:      jobMsg.Data.Status,
		CvGenerated: jobMsg.Data.CvGenerated,
		Cv:          jobMsg.Data.Cv,
		Description: jobMsg.Data.Description,
		CoverLetter: jobMsg.Data.CoverLetter,
		CreatedAt:   jobMsg.Data.CreatedAt,
		AppliedAt:   jobMsg.Data.AppliedAt,
	})
	if err != nil {
		log.Printf("Failed to publish cover letter generated message for job %d: %v", jobMsg.Data.Id, err)
	} else {
		log.Printf("Published cover letter generated message for job %d", jobMsg.Data.Id)
	}

	return nil
}

func handleCoverGenerationRequest(data []byte) error {
	var coverReqMsg CoverGenerationRequestMessage
	if err := json.Unmarshal(data, &coverReqMsg); err != nil {
		log.Printf("Failed to unmarshal cover generation request message: %v", err)
		return err
	}
	log.Printf("Received cover generation request for job ID: %s", coverReqMsg.Data.JobID)

	// Fetch job from database using shared DB
	jobID, err := strconv.Atoi(coverReqMsg.Data.JobID)
	if err != nil {
		log.Printf("Invalid job ID: %s", coverReqMsg.Data.JobID)
		return err
	}

	job, err := sharedDB.GetJobByID(jobID)
	if err == sharedDB.ErrNotFound {
		log.Printf("Job not found for ID: %s", coverReqMsg.Data.JobID)
		return err
	} else if err != nil {
		log.Printf("DB query error: %v", err)
		return err
	}

	var promptText string
	if coverReqMsg.Data.PromptID != nil {
		// Fetch specific prompt
		prompt, err := sharedDB.GetPromptByID(*coverReqMsg.Data.PromptID)
		if err == sharedDB.ErrNotFound {
			log.Printf("Prompt not found for ID %d", *coverReqMsg.Data.PromptID)
			return err
		} else if err != nil {
			log.Printf("Failed to get prompt by ID %d: %v", *coverReqMsg.Data.PromptID, err)
			return err
		}
		promptText = prompt.Prompt
	} else {
		// Get default cover letter generation prompt
		coverPrompt, err := sharedDB.GetDefaultCoverPrompt()
		if err == sharedDB.ErrNotFound {
			log.Printf("No default cover letter generation prompt found")
			promptText = "Generate a professional cover letter for the following job:"
		} else if err != nil {
			log.Printf("Failed to get default cover prompt: %v", err)
			return err
		} else {
			promptText = coverPrompt.Prompt
		}
	}

	// Generate cover letter using AI
	fullPrompt := promptText + "\n\n" +
		"Title: " + job.Title + "\n" +
		"Company: " + job.Company + "\n" +
		"Description: " + job.Description + "\n"

	aiClient, err := sharedAI.DefaultClient()
	if err != nil {
		log.Printf("AI client error: %v", err)
		return err
	}

	coverLetter, err := aiClient.Generate(fullPrompt)
	if err != nil {
		log.Printf("Error generating cover letter: %v", err)
		return err
	}

	log.Printf("Generated cover letter for Job ID: %s", coverReqMsg.Data.JobID)

	// Update job with generated cover letter using shared DB
	jobID, err = strconv.Atoi(coverReqMsg.Data.JobID)
	if err != nil {
		log.Printf("Invalid job ID: %s", coverReqMsg.Data.JobID)
		return err
	}

	err = sharedDB.UpdateJobCoverLetter(jobID, coverLetter)
	if err != nil {
		log.Printf("Failed to update job with generated cover letter: %v", err)
		return err
	}
	log.Printf("Job %s updated with generated cover letter", coverReqMsg.Data.JobID)

	// Publish cover letter generated message using shared NATS
	job.CoverLetter = coverLetter

	err = sharedNats.PublishCoverGeneratedMessage(models.Job{
		Id:          job.Id,
		Title:       job.Title,
		Company:     job.Company,
		Link:        job.Link,
		Status:      job.Status,
		CvGenerated: job.CvGenerated,
		Cv:          job.Cv,
		Description: job.Description,
		CoverLetter: job.CoverLetter,
		CreatedAt:   job.CreatedAt,
		AppliedAt:   job.AppliedAt,
	})
	if err != nil {
		log.Printf("Failed to publish cover letter generated message for job %s: %v", coverReqMsg.Data.JobID, err)
		return err
	} else {
		log.Printf("Published cover letter generated message for job %s", coverReqMsg.Data.JobID)
	}

	return nil
}
