package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

type Job struct {
	Title   string `json:"title"`
	Company string `json:"company"`
	Link    string `json:"link"`
}

var (
	jobs []Job
	mu   sync.Mutex
)

func addJobHandler(w http.ResponseWriter, r *http.Request) {
	// Allow CORS
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
	mu.Lock()
	jobs = append(jobs, job)
	mu.Unlock()
	w.WriteHeader(http.StatusCreated)
}

func listJobsHandler(w http.ResponseWriter, r *http.Request) {
	// Allow CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

func main() {
	http.HandleFunc("/api/jobs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			addJobHandler(w, r)
		} else if r.Method == http.MethodGet {
			listJobsHandler(w, r)
		} else if r.Method == http.MethodOptions {
			// Handle preflight
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	log.Println("Backend running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
