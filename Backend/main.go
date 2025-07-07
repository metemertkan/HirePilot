package main

import (
	"log"
	"net/http"
)

func main() {
	initDB()

	http.HandleFunc("/api/jobs", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			addJobHandler(w, r)
		case http.MethodGet:
			listJobsHandler(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/jobs/", func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) > len("/api/jobs/") &&
			len(r.URL.Path) > len("/apply") &&
			r.URL.Path[len(r.URL.Path)-len("/apply"):] == "/apply" {
			if r.Method == http.MethodPut || r.Method == http.MethodOptions {
				applyJobHandler(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else if len(r.URL.Path) > len("/generate-cv") &&
			r.URL.Path[len(r.URL.Path)-len("/generate-cv"):] == "/generate-cv" {
			generateCVHandler(w, r)
		} else {
			getJobHandler(w, r)
		}
	})

	http.HandleFunc("/api/prompts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			addPromptHandler(w, r)
		case http.MethodGet:
			// Check if "id" query parameter is present for single prompt
			if id := r.URL.Query().Get("id"); id != "" {
				getPromptHandler(w, r)
			} else {
				listPromptsHandler(w, r)
			}
		case http.MethodPut:
			updatePromptHandler(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Println("Backend running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
