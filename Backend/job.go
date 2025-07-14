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
	result, err := db.Exec(
		"INSERT INTO jobs (title, company, link, status, cvGenerated, cv, description) VALUES (?, ?, ?, ?, ?, ?, ?)",
		job.Title, job.Company, job.Link, job.Status, job.CvGenerated, job.Cv, job.Description,
	)
	if err != nil {
		http.Error(w, "DB insert error", http.StatusInternalServerError)
		return
	}
	id, err := result.LastInsertId()
	if err == nil {
		job.Id = int(id)
	}

	//check if feature for cv generation is enabled
	var cvGenerationEnabled bool
	err = db.QueryRow("SELECT value FROM features WHERE name = 'cvGeneration'").
		Scan(&cvGenerationEnabled)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	if cvGenerationEnabled {
		_ = publishJobMessage(job)
	}

	w.WriteHeader(http.StatusCreated)
}
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
	if status == "applied" {
		db.Exec("UPDATE jobs SET applied_at = CURRENT_TIMESTAMP() WHERE id = ?", id)
	}
	_, err := db.Exec("UPDATE jobs SET status = ? WHERE id = ?", status, id)
	if err != nil {
		http.Error(w, "DB update error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
func listJobsByAppliedToday(w http.ResponseWriter, r *http.Request) {
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
	rows, err := db.Query("SELECT count(*) FROM jobs WHERE status = 'applied' AND DATE(applied_at) = CURDATE()")
	if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}
	var count int
	rows.Next()
	rows.Scan(&count)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
	return
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
		rows, err = db.Query("SELECT id, title, company, link, status, cvGenerated, cv, description, score, created_at, applied_at FROM jobs WHERE status = ?", status)
	} else {
		rows, err = db.Query("SELECT id, title, company, link, status, cvGenerated, cv, description, score, created_at, applied_at FROM jobs")
	}
	if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		var job Job
		if err := rows.Scan(&job.Id, &job.Title, &job.Company, &job.Link, &job.Status, &job.CvGenerated, &job.Cv, &job.Description, &job.Score, &job.CreatedAt, &job.AppliedAt); err != nil {
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
	err := db.QueryRow("SELECT id, title, company, link, status, cvGenerated, cv, description, score, created_at, applied_at FROM jobs WHERE id = ?", id).
		Scan(&job.Id, &job.Title, &job.Company, &job.Link, &job.Status, &job.CvGenerated, &job.Cv, &job.Description, &job.Score, &job.CreatedAt, &job.AppliedAt)
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
