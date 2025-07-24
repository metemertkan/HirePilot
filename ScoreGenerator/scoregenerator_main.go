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

type ScoreGenerationRequestMessage struct {
	Type string                            `json:"type"`
	Data sharedNats.ScoreGenerationRequest `json:"data"`
}

func main() {
	// Initialize shared database
	sharedDB.InitDB()

	// Initialize shared NATS JetStream
	sharedNats.InitJetStream()

	if _, err := sharedNats.SubscribeToCVGeneratedForScore(handleCVGenerated); err != nil {
		log.Fatalf("Failed to start CV created consumer: %v", err)
	}

	if _, err := sharedNats.SubscribeToScoreGenerationRequestsGeneric(handleScoreGenerationRequest); err != nil {
		log.Fatalf("Failed to start score generation request consumer: %v", err)
	}

	log.Println("ScoreGenerator started successfully")
	// Block forever
	select {}
}

func handleCVGenerated(data []byte) error {
	var jobMsg JobMessage
	if err := json.Unmarshal(data, &jobMsg); err != nil {
		log.Printf("Failed to unmarshal job message: %v", err)
		return err
	}
	log.Printf("Received job: %+v", jobMsg.Data.Id)

	// Get the default score generation prompt from database
	scorePromptObj, err := sharedDB.GetDefaultScorePrompt()
	if err == sharedDB.ErrNotFound {
		log.Printf("No default score generation prompt found")
		return fmt.Errorf("no default score generation prompt found")
	} else if err != nil {
		log.Printf("Failed to get default score prompt: %v", err)
		return err
	}

	scorePrompt := scorePromptObj.Prompt + "\n\n" +
		"Job Description: " + jobMsg.Data.Description + "\n" +
		"CV: " + jobMsg.Data.Cv + "\n"

	// Use shared AI client to generate score
	aiClient, err := sharedAI.DefaultClient()
	if err != nil {
		log.Printf("AI client error: %v", err)
		return err
	}

	score, err := aiClient.Generate(scorePrompt)
	if err != nil {
		log.Printf("AI generation error: %v", err)
		return err
	}
	log.Printf("Generated Score : %s", score)

	// Update job with score using shared DB
	err = sharedDB.UpdateJobScore(jobMsg.Data.Id, score)
	if err != nil {
		log.Printf("DB update error: %v", err)
		return err
	}
	log.Printf("Job %d updated with generated Score", jobMsg.Data.Id)
	return nil
}

func handleScoreGenerationRequest(data []byte) error {
	var scoreReqMsg ScoreGenerationRequestMessage
	if err := json.Unmarshal(data, &scoreReqMsg); err != nil {
		log.Printf("Failed to unmarshal score generation request message: %v", err)
		return err
	}
	log.Printf("Received score generation request for job ID: %s", scoreReqMsg.Data.JobID)

	// Fetch job from database using shared DB
	jobID, err := strconv.Atoi(scoreReqMsg.Data.JobID)
	if err != nil {
		log.Printf("Invalid job ID: %s", scoreReqMsg.Data.JobID)
		return err
	}

	job, err := sharedDB.GetJobByID(jobID)
	if err == sharedDB.ErrNotFound {
		log.Printf("Job not found for ID: %s", scoreReqMsg.Data.JobID)
		return err
	} else if err != nil {
		log.Printf("Failed to fetch job for ID %s: %v", scoreReqMsg.Data.JobID, err)
		return err
	}

	// Check if CV exists
	if !job.CvGenerated || job.Cv == "" {
		log.Printf("Job %s does not have a CV generated yet", scoreReqMsg.Data.JobID)
		return fmt.Errorf("job %s does not have a CV generated yet", scoreReqMsg.Data.JobID)
	}

	var promptText string
	if scoreReqMsg.Data.PromptID != nil {
		// Fetch specific prompt
		prompt, err := sharedDB.GetPromptByID(*scoreReqMsg.Data.PromptID)
		if err == sharedDB.ErrNotFound {
			log.Printf("Prompt not found for ID %d", *scoreReqMsg.Data.PromptID)
			return err
		} else if err != nil {
			log.Printf("Failed to get prompt by ID %d: %v", *scoreReqMsg.Data.PromptID, err)
			return err
		}
		promptText = prompt.Prompt
	} else {
		// Get default score generation prompt
		scorePromptObj, err := sharedDB.GetDefaultScorePrompt()
		if err == sharedDB.ErrNotFound {
			log.Printf("No default score generation prompt found")
			promptText = "Score the following CV based on the provided job description. The CV will start below '**Resume**'. Only return a numerical score between 0 and 100, where 0 is a poor match and 100 is a perfect match. Do not include any explanations, notes, or additional text."
		} else if err != nil {
			log.Printf("Failed to get default score prompt: %v", err)
			return err
		} else {
			promptText = scorePromptObj.Prompt
		}
	}

	// Generate score using AI
	scorePrompt := promptText + "\n\n" +
		"Job Description: " + job.Description + "\n" +
		"CV: " + job.Cv + "\n"

	aiClient, err := sharedAI.DefaultClient()
	if err != nil {
		log.Printf("AI client error: %v", err)
		return err
	}

	score, err := aiClient.Generate(scorePrompt)
	if err != nil {
		log.Printf("AI generation error: %v", err)
		return err
	}

	log.Printf("Generated score for Job ID: %s - Score: %s", scoreReqMsg.Data.JobID, score)

	// Update job with generated score using shared DB
	jobID, err = strconv.Atoi(scoreReqMsg.Data.JobID)
	if err != nil {
		log.Printf("Invalid job ID: %s", scoreReqMsg.Data.JobID)
		return err
	}

	err = sharedDB.UpdateJobScore(jobID, score)
	if err != nil {
		log.Printf("Failed to update job with generated score: %v", err)
		return err
	}
	log.Printf("Job %s updated with generated score", scoreReqMsg.Data.JobID)

	return nil
}
