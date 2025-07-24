package models

import (
	"time"
)

type JobStatus string

const (
	JobStatusOpen    JobStatus = "open"
	JobStatusApplied JobStatus = "applied"
	JobStatusClosed  JobStatus = "closed"
)

// Job represents the job structure shared across all services
type Job struct {
	Id          int        `json:"id" db:"id"`
	Title       string     `json:"title" db:"title"`
	Company     string     `json:"company" db:"company"`
	Link        string     `json:"link" db:"link"`
	Status      JobStatus  `json:"status" db:"status"`
	CvGenerated bool       `json:"cvGenerated" db:"cvGenerated"`
	Cv          string     `json:"cv" db:"cv"`
	Description string     `json:"description" db:"description"`
	Score       *float64   `json:"score" db:"score"`
	AppliedAt   *time.Time `json:"applied_at" db:"applied_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	CoverLetter string     `json:"cover_letter" db:"cover_letter"`
}

// Prompt represents the prompt structure shared across all services
type Prompt struct {
	Id                      int    `json:"id" db:"id"`
	Name                    string `json:"name" db:"name"`
	Prompt                  string `json:"prompt" db:"prompt"`
	CvGenerationDefault     bool   `json:"cvGenerationDefault" db:"cvGenerationDefault"`
	ScoreGenerationDefault  bool   `json:"scoreGenerationDefault" db:"scoreGenerationDefault"`
	CoverGenerationDefault  bool   `json:"coverGenerationDefault" db:"coverGenerationDefault"`
}

// Feature represents the feature structure shared across all services
type Feature struct {
	Id    int    `json:"id" db:"id"`
	Name  string `json:"name" db:"name"`
	Value bool   `json:"value" db:"value"`
}