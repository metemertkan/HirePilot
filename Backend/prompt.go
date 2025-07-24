package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	
	sharedDB "github.com/hirepilot/shared/db"
	"github.com/hirepilot/shared/models"
	sharedNats "github.com/hirepilot/shared/nats"
)

func addPromptHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "POST, OPTIONS")
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	var prompt models.Prompt
	if err := json.NewDecoder(r.Body).Decode(&prompt); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Publish prompt creation request to NATS JetStream (PromptService will handle DB insertion)
	err := sharedNats.PublishPromptCreationRequest(prompt.Name, prompt.Prompt, prompt.CvGenerationDefault, prompt.ScoreGenerationDefault, prompt.CoverGenerationDefault)
	if err != nil {
		log.Printf("Failed to publish prompt creation request: %v", err)
		http.Error(w, "Failed to process prompt creation request", http.StatusInternalServerError)
		return
	}

	log.Printf("Prompt creation request published for: %s", prompt.Name)
	w.WriteHeader(http.StatusAccepted) // 202 Accepted since processing is async
}

func listPromptsHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "GET, OPTIONS")
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}

	prompts, err := sharedDB.GetAllPrompts()
	if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(prompts); err != nil {
		http.Error(w, "JSON encode error", http.StatusInternalServerError)
		return
	}
}

func updatePromptHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "PUT, OPTIONS")
		return
	}
	if r.Method != http.MethodPut {
		http.Error(w, "Only PUT allowed", http.StatusMethodNotAllowed)
		return
	}

	var prompt models.Prompt
	if err := json.NewDecoder(r.Body).Decode(&prompt); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Publish prompt update request to NATS JetStream (PromptService will handle DB update)
	err := sharedNats.PublishPromptUpdateRequest(prompt.Id, prompt.Name, prompt.Prompt, prompt.CvGenerationDefault, prompt.ScoreGenerationDefault, prompt.CoverGenerationDefault)
	if err != nil {
		log.Printf("Failed to publish prompt update request: %v", err)
		http.Error(w, "Failed to process prompt update request", http.StatusInternalServerError)
		return
	}

	log.Printf("Prompt update request published for ID: %d", prompt.Id)
	w.WriteHeader(http.StatusAccepted) // 202 Accepted since processing is async
}

func getPromptHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "GET, OPTIONS")
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse ID from query parameters
	ids, ok := r.URL.Query()["id"]
	if !ok || len(ids[0]) < 1 {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	// Convert id to int
	id, err := strconv.Atoi(ids[0])
	if err != nil {
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	prompt, err := sharedDB.GetPromptByID(id)
	if err == sharedDB.ErrNotFound {
		http.Error(w, "Prompt not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(prompt); err != nil {
		http.Error(w, "JSON encode error", http.StatusInternalServerError)
		return
	}
}
