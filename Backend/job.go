package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
)

func addJobHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	var job Job
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	_, err := db.Exec(
		"INSERT INTO jobs (title, company, link, status, cvGenerated, cv, description) VALUES (?, ?, ?, ?, ?, ?, ?)",
		job.Title, job.Company, job.Link, job.Status, job.CvGenerated, job.Cv, job.Description,
	)
	if err != nil {
		http.Error(w, "DB insert error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// Handler to update job status (e.g., 'applied', 'closed')
func updateJobStatusHandler(w http.ResponseWriter, r *http.Request, status string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPut {
		http.Error(w, "Only PUT allowed", http.StatusMethodNotAllowed)
		return
	}
	// Extract id from URL: /api/jobs/{id}/apply or /api/jobs/{id}/close
	idWithAction := r.URL.Path[len("/api/jobs/"):] // e.g. "123/apply" or "123/close"
	var actionSuffix string
	if status == "applied" {
		actionSuffix = "/apply"
	} else if status == "closed" {
		actionSuffix = "/close"
	} else if status == "open" {
		actionSuffix = "/open"
	} else {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}
	id := idWithAction[:len(idWithAction)-len(actionSuffix)]
	_, err := db.Exec("UPDATE jobs SET status = ? WHERE id = ?", status, id)
	if err != nil {
		http.Error(w, "DB update error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
func listJobsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	status := r.URL.Query().Get("status")
	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = db.Query("SELECT id, title, company, link, status, cvGenerated, cv, description, score FROM jobs WHERE status = ?", status)
	} else {
		rows, err = db.Query("SELECT id, title, company, link, status, cvGenerated, cv, description, score FROM jobs")
	}
	if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		var job Job
		if err := rows.Scan(&job.Id, &job.Title, &job.Company, &job.Link, &job.Status, &job.CvGenerated, &job.Cv, &job.Description, &job.Score); err != nil {
			http.Error(w, "DB scan error", http.StatusInternalServerError)
			return
		}
		jobs = append(jobs, job)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}
func getJobHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}
	// Extract id from URL: /api/jobs/{id}
	id := r.URL.Path[len("/api/jobs/"):]
	var job Job
	err := db.QueryRow("SELECT id, title, company, link, status, cvGenerated, cv, description, score FROM jobs WHERE id = ?", id).
		Scan(&job.Id, &job.Title, &job.Company, &job.Link, &job.Status, &job.CvGenerated, &job.Cv, &job.Description, &job.Score)
	if err == sql.ErrNoRows {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func generateScoreHandler(w http.ResponseWriter, r *http.Request) {
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

	var promptText string
	if req.PromptId != nil {
		err := db.QueryRow("SELECT prompt FROM prompts WHERE id = ?", *req.PromptId).Scan(&promptText)
		if err == sql.ErrNoRows {
			http.Error(w, "Prompt not found", http.StatusBadRequest)
			return
		} else if err != nil {
			http.Error(w, "DB error fetching prompt", http.StatusInternalServerError)
			return
		}
	} else {
		// Fallback to default prompt if not provided
		promptText = "Score the following CV based on the provided job description. The CV will start below '**Resume**'. Only return a numerical score between 0 and 100, where 0 is a poor match and 100 is a perfect match. Do not include any explanations, notes, or additional text."
	}

	// Always append job title, company, and description to the prompt
	scorePrompt := promptText + "\n\n" +
		"Job Description: " + job.Description + "\n" +
		"CV: " + job.Cv + "\n"

	score, err := generateCVWithGemini(scorePrompt)
	if err != nil {
		http.Error(w, "Gemini API error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = db.Exec("UPDATE jobs SET score = ? WHERE id = ?", score, id)
	if err != nil {
		http.Error(w, "DB update error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	scoreValue, err := strconv.ParseFloat(score, 64)
	if err != nil {
		http.Error(w, "Cannot convert to float", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]float64{"score": scoreValue})
}
