package main

import (
	"encoding/json"
	"net/http"

	sharedNATS "github.com/hirepilot/shared/nats"
)

func generateCVHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "POST, OPTIONS")
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	// Extract id from URL: /api/jobs/{id}/generate-cv
	path := r.URL.Path
	prefix := "/api/jobs/"
	suffix := "/generate-cv"
	if !(len(path) > len(prefix)+len(suffix) && path[len(path)-len(suffix):] == suffix) {
		http.Error(w, "Invalid endpoint", http.StatusBadRequest)
		return
	}
	id := path[len(prefix) : len(path)-len(suffix)]

	// Parse promptId from request body
	var reqBody struct {
		PromptId *int `json:"promptId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Send CV generation request via NATS instead of generating directly
	err := sharedNATS.PublishCVGenerationRequest(id, reqBody.PromptId)
	if err != nil {
		http.Error(w, "Failed to publish CV generation request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response indicating CV generation has been requested
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "CV generation requested",
		"job_id":  id,
		"status":  "processing",
	})
}
func generateScoreHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "POST, OPTIONS")
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	// Extract id from URL: /api/jobs/{id}/generate-score
	idWithAction := r.URL.Path[len("/api/jobs/"):] // e.g. "123/generate-score"
	id := idWithAction[:len(idWithAction)-len("/generate-score")]
	var req struct {
		PromptId *int `json:"promptId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Send score generation request via NATS instead of generating directly
	err := sharedNATS.PublishScoreGenerationRequest(id, req.PromptId)
	if err != nil {
		http.Error(w, "Failed to publish score generation request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response indicating score generation has been requested
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Score generation requested",
		"job_id":  id,
		"status":  "processing",
	})
}
