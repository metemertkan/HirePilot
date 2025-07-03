package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Job struct {
	Id          int    `json:"id"`
	Title       string `json:"title"`
	Company     string `json:"company"`
	Link        string `json:"link"`
	Applied     bool   `json:"applied"`
	CvGenerated bool   `json:"cvGenerated"`
	Cv          string `json:"cv"`
}

var (
	db *sql.DB
	mu sync.Mutex
)

func initDB() {
	dsn := os.Getenv("DB_USER") + ":" + os.Getenv("DB_PASSWORD") +
		"@tcp(" + os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT") + ")/" + os.Getenv("DB_NAME")
	var err error
	for i := 0; i < 10; i++ { // Try 10 times
		db, err = sql.Open("mysql", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Waiting for DB to be ready (%d/10): %v", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("DB ping error: %v", err)
	}
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS jobs (
            id INT AUTO_INCREMENT PRIMARY KEY,
            title VARCHAR(255) NOT NULL,
            company VARCHAR(255) NOT NULL,
            link VARCHAR(512) NOT NULL,
            applied BOOLEAN DEFAULT FALSE,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            cvGenerated BOOLEAN DEFAULT FALSE,
            cv TEXT
        )
    `)
	if err != nil {
		log.Fatalf("Table creation error: %v", err)
	}
}

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
		"INSERT INTO jobs (title, company, link, applied, cvGenerated, cv) VALUES (?, ?, ?, ?, ?, ?)",
		job.Title, job.Company, job.Link, job.Applied, job.CvGenerated, job.Cv,
	)
	if err != nil {
		http.Error(w, "DB insert error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func listJobsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	rows, err := db.Query("SELECT id, title, company, link, applied, cvGenerated, cv FROM jobs")
	if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		var job Job
		if err := rows.Scan(&job.Id, &job.Title, &job.Company, &job.Link, &job.Applied, &job.CvGenerated, &job.Cv); err != nil {
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
	err := db.QueryRow("SELECT id, title, company, link, applied, cvGenerated, cv FROM jobs WHERE id = ?", id).
		Scan(&job.Id, &job.Title, &job.Company, &job.Link, &job.Applied, &job.CvGenerated, &job.Cv)
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
	err := db.QueryRow("SELECT id, title, company, link, applied, cvGenerated, cv FROM jobs WHERE id = ?", id).
		Scan(&job.Id, &job.Title, &job.Company, &job.Link, &job.Applied, &job.CvGenerated, &job.Cv)
	if err == sql.ErrNoRows {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	// Generate CV (placeholder)
	cv := "Generated CV for " + job.Title + " at " + job.Company

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
			len(r.URL.Path) > len("/generate-cv") &&
			r.URL.Path[len(r.URL.Path)-len("/generate-cv"):] == "/generate-cv" {
			generateCVHandler(w, r)
		} else {
			getJobHandler(w, r)
		}
	})
	log.Println("Backend running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
