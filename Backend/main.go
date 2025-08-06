package main

import (
	"log"
	"net/http"

	sharedDB "github.com/hirepilot/shared/db"
	sharedNats "github.com/hirepilot/shared/nats"
)

// CORS helper functions
func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func handleCORS(w http.ResponseWriter, methods string) {
	setCORSHeaders(w)
	w.Header().Set("Access-Control-Allow-Methods", methods)
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	// Initialize shared database for read operations
	sharedDB.InitDB()

	// Initialize shared NATS JetStream
	sharedNats.InitJetStream()

	http.HandleFunc("/api/jobs", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			addJobHandler(w, r)
		case http.MethodGet:
			listJobsHandler(w, r)
		case http.MethodOptions:
			handleCORS(w, "POST, GET, OPTIONS")
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/jobs/", func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) > len("/api/jobs/") &&
			len(r.URL.Path) > len("/apply") &&
			r.URL.Path[len(r.URL.Path)-len("/apply"):] == "/apply" {
			if r.Method == http.MethodPut || r.Method == http.MethodOptions {
				updateJobStatusHandler(w, r, "applied")
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else if len(r.URL.Path) > len("/api/jobs/") &&
			len(r.URL.Path) > len("/close") &&
			r.URL.Path[len(r.URL.Path)-len("/close"):] == "/close" {
			if r.Method == http.MethodPut || r.Method == http.MethodOptions {
				updateJobStatusHandler(w, r, "closed")
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else if len(r.URL.Path) > len("/generate-cv") &&
			r.URL.Path[len(r.URL.Path)-len("/generate-cv"):] == "/generate-cv" {
			generateCVHandler(w, r)
		} else if len(r.URL.Path) > len("/generate-score") &&
			r.URL.Path[len(r.URL.Path)-len("/generate-score"):] == "/generate-score" {
			generateScoreHandler(w, r)
		} else if len(r.URL.Path) > len("/regenerate") &&
			r.URL.Path[len(r.URL.Path)-len("/regenerate"):] == "/regenerate" {
			regenerateJobContentHandler(w, r)
		} else if len(r.URL.Path) > len("/today") &&
			r.URL.Path[len(r.URL.Path)-len("/today"):] == "/today" {
			listJobsByAppliedToday(w, r)
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
			handleCORS(w, "POST, GET, PUT, OPTIONS")
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/features", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("name") != "" {
				getFeatureHandler(w, r)
			} else {
				listFeaturesHandler(w, r)
			}
		case http.MethodPut:
			updateFeatureHandler(w, r)
		case http.MethodOptions:
			handleCORS(w, "GET, PUT, OPTIONS")
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// WebSocket endpoint for real-time updates
	log.Println("Backend running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
