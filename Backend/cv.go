package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

func generateCVHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
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

	// Fetch job
	var job Job
	err := db.QueryRow("SELECT id, title, company, link, status, cvGenerated, cv, description FROM jobs WHERE id = ?", id).
		Scan(&job.Id, &job.Title, &job.Company, &job.Link, &job.Status, &job.CvGenerated, &job.Cv, &job.Description)
	if err == sql.ErrNoRows {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	// Parse promptId from request body
	var reqBody struct {
		PromptId *int `json:"promptId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var promptText string
	if reqBody.PromptId != nil {
		// Fetch prompt text from DB
		err := db.QueryRow("SELECT prompt FROM prompts WHERE id = ?", *reqBody.PromptId).Scan(&promptText)
		if err == sql.ErrNoRows {
			http.Error(w, "Prompt not found", http.StatusBadRequest)
			return
		} else if err != nil {
			http.Error(w, "DB error fetching prompt", http.StatusInternalServerError)
			return
		}
	} else {
		// Fallback to default prompt if not provided
		promptText = "Generate a professional CV for the following job:"
	}

	// Always append job title, company, and description to the prompt
	cvPrompt := promptText + "\n\n" +
		"Title: " + job.Title + "\n" +
		"Company: " + job.Company + "\n" +
		"Description: " + job.Description + "\n"

	cv, err := generateCVWithGemini(cvPrompt)
	if err != nil {
		http.Error(w, "Gemini API error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Update job in DB
	_, err = db.Exec("UPDATE jobs SET cvGenerated = ?, cv = ? WHERE id = ?", true, cv, id)
	if err != nil {
		http.Error(w, "DB update error", http.StatusInternalServerError)
		return
	}

	// Return updated job
	job.CvGenerated = true
	job.Cv = cv
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}
