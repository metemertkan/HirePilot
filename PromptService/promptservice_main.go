package main

import (
	"encoding/json"
	"log"

	sharedDB "github.com/hirepilot/shared/db"
	sharedNats "github.com/hirepilot/shared/nats"
)

type PromptCreationMessage struct {
	Type string                           `json:"type"`
	Data sharedNats.PromptCreationRequest `json:"data"`
}

type PromptUpdateMessage struct {
	Type string                         `json:"type"`
	Data sharedNats.PromptUpdateRequest `json:"data"`
}

func main() {
	log.Println("Starting Prompt Service...")

	// Initialize shared database
	sharedDB.InitDB()

	// Initialize shared NATS JetStream
	sharedNats.InitJetStream()
	defer sharedNats.Close()

	// Subscribe to prompt creation requests
	_, err := sharedNats.SubscribeToPromptCreationRequestsGeneric(func(data []byte) error {
		log.Printf("Received prompt creation request")

		// Parse the message
		var message PromptCreationMessage
		if err := json.Unmarshal(data, &message); err != nil {
			log.Printf("Error unmarshaling prompt creation message: %v", err)
			return err
		}

		// Process the prompt creation request
		if err := handlePromptCreation(message.Data); err != nil {
			log.Printf("Error handling prompt creation: %v", err)
			return err
		}

		log.Printf("Prompt creation handled successfully")
		return nil
	})

	if err != nil {
		log.Fatalf("Failed to subscribe to prompt creation messages: %v", err)
	}

	// Subscribe to prompt update requests
	_, err = sharedNats.SubscribeToPromptUpdateRequestsGeneric(func(data []byte) error {
		log.Printf("Received prompt update request")

		var message PromptUpdateMessage
		if err := json.Unmarshal(data, &message); err != nil {
			log.Printf("Error unmarshaling prompt update message: %v", err)
			return err
		}

		// Process the prompt update request
		if err := handlePromptUpdate(message.Data); err != nil {
			log.Printf("Error handling prompt update: %v", err)
			return err
		}

		log.Printf("Prompt update handled successfully")
		return nil
	})

	if err != nil {
		log.Fatalf("Failed to subscribe to prompt update messages: %v", err)
	}

	log.Println("Prompt Service subscribed to prompt creation and update messages")
	log.Println("Prompt Service is running. Press Ctrl+C to exit.")

	// Keep the service running
	select {}
}

func handlePromptCreation(promptData sharedNats.PromptCreationRequest) error {
	log.Printf("Processing prompt creation: %s", promptData.Name)

	// Insert prompt into database using shared library
	id, err := sharedDB.InsertPromptWithCover(promptData.Name, promptData.Prompt, promptData.CvGenerationDefault, promptData.ScoreGenerationDefault, promptData.CoverGenerationDefault)
	if err != nil {
		return err
	}

	log.Printf("Prompt saved to database with ID: %d", id)
	return nil
}

func handlePromptUpdate(promptData sharedNats.PromptUpdateRequest) error {
	log.Printf("Processing prompt update: ID %d", promptData.ID)

	// Update prompt in database using shared library
	err := sharedDB.UpdatePromptWithCover(promptData.ID, promptData.Name, promptData.Prompt, promptData.CvGenerationDefault, promptData.ScoreGenerationDefault, promptData.CoverGenerationDefault)
	if err != nil {
		return err
	}

	log.Printf("Prompt %d updated in database", promptData.ID)
	return nil
}
