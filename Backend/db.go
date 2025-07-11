package main

import (
	"database/sql"
	"log"
	"os"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

type JobStatus string

const (
	JobStatusOpen    JobStatus = "open"
	JobStatusApplied JobStatus = "applied"
	JobStatusClosed  JobStatus = "closed"
)

type Job struct {
	Id          int       `json:"id"`
	Title       string    `json:"title"`
	Company     string    `json:"company"`
	Link        string    `json:"link"`
	Status      JobStatus `json:"status"`
	CvGenerated bool      `json:"cvGenerated"`
	Cv          string    `json:"cv"`
	Description string    `json:"description"`
	Score       *float64  `json:"score"`
}

type Prompt struct {
	Id                     int    `json:"id"`
	Name                   string `json:"name"`
	Prompt                 string `json:"prompt"`
	CvGenerationDefault    string `json:"cvGenerationDefault"`
	ScoreGenerationDefault string `json:"scoreGenerationDefault"`
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
		// Sleep for 3 seconds before retrying
		// (import "time" in main.go for this)
	}
	if err != nil {
		log.Fatalf("DB ping error: %v", err)
	}
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS jobs (
	id INT AUTO_INCREMENT PRIMARY KEY,
	title VARCHAR(255),
	company VARCHAR(255),
	link VARCHAR(512),
	status ENUM('open','applied','closed') NOT NULL DEFAULT 'open',
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	cvGenerated BOOLEAN DEFAULT FALSE,
	cv TEXT,
	description TEXT,
	score FLOAT NULL
	)
	`)
	if err != nil {
		log.Fatalf("Table creation error: %v", err)
	}
	// Create prompts table
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS prompts (
	id INT AUTO_INCREMENT PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	prompt TEXT,
	cvGenerationDefault BOOLEAN DEFAULT FALSE,
	scoreGenerationDefault BOOLEAN DEFAULT FALSE
	)
	`)
	if err != nil {
		log.Fatalf("Prompts table creation error: %v", err)
	}
}
