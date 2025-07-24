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

func addJobHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "POST, GET, OPTIONS")
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var job models.Job
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Publish job creation request to NATS JetStream (JobService will handle DB insertion)
	err := sharedNats.PublishJobCreationRequest(job.Title, job.Company, job.Link, job.Description)
	if err != nil {
		log.Printf("Failed to publish job creation request: %v", err)
		http.Error(w, "Failed to process job creation request", http.StatusInternalServerError)
		return
	}

	log.Printf("Job creation request published for: %s at %s", job.Title, job.Company)
	w.WriteHeader(http.StatusAccepted) // 202 Accepted since processing is async
}
func updateJobStatusHandler(w http.ResponseWriter, r *http.Request, status string) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "PUT, OPTIONS")
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
	idStr := idWithAction[:len(idWithAction)-len(actionSuffix)]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	// Send job status update request via NATS instead of updating directly
	err = sharedNats.PublishJobStatusUpdateRequest(id, status)
	if err != nil {
		http.Error(w, "Failed to publish job status update request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response indicating status update has been requested
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Job status update requested",
		"job_id":  id,
		"status":  status,
	})
}
func listJobsByAppliedToday(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "GET, OPTIONS")
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}

	count, err := sharedDB.GetAppliedJobsToday()
	if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}
func listJobsHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "POST, GET, OPTIONS")
		return
	}

	status := r.URL.Query().Get("status")
	jobs, err := sharedDB.GetJobsByStatus(status)
	if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}
func getJobHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "GET, OPTIONS")
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}
	// Extract id from URL: /api/jobs/{id}
	idStr := r.URL.Path[len("/api/jobs/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	job, err := sharedDB.GetJobByID(id)
	if err == sharedDB.ErrNotFound {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func regenerateJobContentHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "POST, OPTIONS")
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract id from URL: /api/jobs/{id}/regenerate
	idWithAction := r.URL.Path[len("/api/jobs/"):] // e.g. "123/regenerate"
	actionSuffix := "/regenerate"
	idStr := idWithAction[:len(idWithAction)-len(actionSuffix)]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	// Parse request body for prompt ID
	var requestBody struct {
		PromptID int `json:"promptId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Get the job from database
	_, err = sharedDB.GetJobByID(id)
	if err == sharedDB.ErrNotFound {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	// Publish CV generation request
	err = sharedNats.PublishCVGenerationRequest(strconv.Itoa(id), &requestBody.PromptID)
	if err != nil {
		log.Printf("Failed to publish CV generation request: %v", err)
		http.Error(w, "Failed to publish CV generation request", http.StatusInternalServerError)
		return
	}

	// Publish cover letter generation request
	err = sharedNats.PublishCoverGenerationRequest(strconv.Itoa(id), &requestBody.PromptID)
	if err != nil {
		log.Printf("Failed to publish cover generation request: %v", err)
		http.Error(w, "Failed to publish cover generation request", http.StatusInternalServerError)
		return
	}

	log.Printf("Regeneration requests published for job %d", id)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Regeneration requests published",
		"job_id":  id,
	})
}
