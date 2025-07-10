package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
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
	rows, err := db.Query("SELECT id, title, company, link, status, cvGenerated, cv, description FROM jobs")
	if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		var job Job
		if err := rows.Scan(&job.Id, &job.Title, &job.Company, &job.Link, &job.Status, &job.CvGenerated, &job.Cv, &job.Description); err != nil {
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
	err := db.QueryRow("SELECT id, title, company, link, status, cvGenerated, cv, description FROM jobs WHERE id = ?", id).
		Scan(&job.Id, &job.Title, &job.Company, &job.Link, &job.Status, &job.CvGenerated, &job.Cv, &job.Description)
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
