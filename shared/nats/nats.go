package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/hirepilot/shared/models"
)

var (
	js   jetstream.JetStream
	nc   *nats.Conn
	once sync.Once
)

// MessageHandler represents a generic message handler function
type MessageHandler func(data []byte) error

// Using shared models package for Job type

// CVData represents CV generation data
type CVData struct {
	JobID       int    `json:"id"`
	Title       string `json:"title"`
	Company     string `json:"company"`
	CVContent   string `json:"cv"`
	Description string `json:"description"`
}

// InitJetStream initializes the JetStream connection (singleton)
func InitJetStream() jetstream.JetStream {
	once.Do(func() {
		// Get NATS URL from environment variable, default to localhost
		natsURL := os.Getenv("NATS_URL")
		if natsURL == "" {
			natsURL = "nats://localhost:4222"
		}

		// Connect to NATS
		var err error
		nc, err = nats.Connect(natsURL)
		if err != nil {
			log.Printf("Failed to connect to NATS: %v", err)
			return
		}

		js, err = jetstream.New(nc)
		if err != nil {
			log.Printf("Failed to create JetStream context: %v", err)
			return
		}

		// Create JOBS stream
		createJobsStream()
		
		log.Println("JetStream initialized successfully")
	})
	
	return js
}

// GetJetStream returns the existing JetStream instance
func GetJetStream() jetstream.JetStream {
	if js == nil {
		return InitJetStream()
	}
	return js
}

// Close closes the NATS connection
func Close() {
	if nc != nil {
		nc.Close()
	}
}

// createJobsStream creates the JOBS stream with all necessary subjects
func createJobsStream() {
	stream, err := js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:      "JOBS",
		Subjects:  []string{"jobs.*", "cv.*", "cover.*"},
		Retention: jetstream.LimitsPolicy,
		MaxAge:    24 * time.Hour, // Keep messages for 24 hours
	})

	if err != nil {
		log.Printf("Failed to create JOBS stream: %v", err)
	} else {
		info, err := stream.Info(context.Background())
		if err != nil {
			log.Printf("Failed to get stream info: %v", err)
		} else {
			log.Printf("Created/Updated stream: %s", info.Config.Name)
		}
	}
}

// PublishJobCreationRequest publishes a job creation request (before DB insertion)
func PublishJobCreationRequest(title, company, link, description string) error {
	js := GetJetStream()
	if js == nil {
		return fmt.Errorf("JetStream not initialized")
	}

	// Create message payload for job creation request
	message := map[string]interface{}{
		"type": "job_creation_request",
		"data": map[string]interface{}{
			"title":       title,
			"company":     company,
			"link":        link,
			"description": description,
		},
	}

	// Convert to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish to JetStream
	_, err = js.Publish(context.Background(), "jobs.create_request", payload)
	if err != nil {
		return err
	}

	log.Printf("Published job creation request for: %s at %s", title, company)
	return nil
}

// PublishJobMessage publishes a job creation message (after DB insertion)
func PublishJobMessage(job models.Job) error {
	js := GetJetStream()
	if js == nil {
		return fmt.Errorf("JetStream not initialized")
	}

	// Create message payload
	message := map[string]interface{}{
		"type": "job_created",
		"data": job,
	}

	// Convert to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish to JetStream
	_, err = js.Publish(context.Background(), "jobs.created", payload)
	if err != nil {
		return err
	}

	log.Printf("Published job message for job ID: %d", job.Id)
	return nil
}

// PublishCVMessage publishes a CV generation completion message
func PublishCVMessage(cvData CVData) error {
	js := GetJetStream()
	if js == nil {
		return fmt.Errorf("JetStream not initialized")
	}

	// Create message payload
	message := map[string]interface{}{
		"type": "cv_generated",
		"data": cvData,
	}

	// Convert to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish to JetStream
	_, err = js.Publish(context.Background(), "cv.generated", payload)
	if err != nil {
		return err
	}

	log.Printf("Published CV message for job ID: %d", cvData.JobID)
	return nil
}

// PublishJobUpdateMessage publishes a job update message
func PublishJobUpdateMessage(jobID string, updates map[string]interface{}) error {
	js := GetJetStream()
	if js == nil {
		return fmt.Errorf("JetStream not initialized")
	}

	// Create message payload
	message := map[string]interface{}{
		"type": "job_update",
		"data": map[string]interface{}{
			"job_id": jobID,
			"updates": updates,
		},
	}

	// Convert to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish to JetStream
	_, err = js.Publish(context.Background(), "jobs.update", payload)
	if err != nil {
		return err
	}

	log.Printf("Published job update message for job ID: %s", jobID)
	return nil
}

// CVGenerationRequest represents a CV generation request
type CVGenerationRequest struct {
	JobID    string `json:"job_id"`
	PromptID *int   `json:"prompt_id,omitempty"`
}

// PublishCVGenerationRequest publishes a CV generation request message
func PublishCVGenerationRequest(jobID string, promptID *int) error {
	js := GetJetStream()
	if js == nil {
		return fmt.Errorf("JetStream not initialized")
	}

	// Create message payload
	message := map[string]interface{}{
		"type": "cv_generation_request",
		"data": CVGenerationRequest{
			JobID:    jobID,
			PromptID: promptID,
		},
	}

	// Convert to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish to JetStream
	_, err = js.Publish(context.Background(), "cv.generate_request", payload)
	if err != nil {
		return err
	}

	log.Printf("Published CV generation request for job ID: %s", jobID)
	return nil
}

// ScoreGenerationRequest represents a score generation request
type ScoreGenerationRequest struct {
	JobID    string `json:"job_id"`
	PromptID *int   `json:"prompt_id,omitempty"`
}

// PublishScoreGenerationRequest publishes a score generation request message
func PublishScoreGenerationRequest(jobID string, promptID *int) error {
	js := GetJetStream()
	if js == nil {
		return fmt.Errorf("JetStream not initialized")
	}

	// Create message payload
	message := map[string]interface{}{
		"type": "score_generation_request",
		"data": ScoreGenerationRequest{
			JobID:    jobID,
			PromptID: promptID,
		},
	}

	// Convert to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish to JetStream
	_, err = js.Publish(context.Background(), "score.generate_request", payload)
	if err != nil {
		return err
	}

	log.Printf("Published score generation request for job ID: %s", jobID)
	return nil
}

// CoverGenerationRequest represents a cover letter generation request
type CoverGenerationRequest struct {
	JobID    string `json:"job_id"`
	PromptID *int   `json:"prompt_id,omitempty"`
}

// PublishCoverGenerationRequest publishes a cover letter generation request message
func PublishCoverGenerationRequest(jobID string, promptID *int) error {
	js := GetJetStream()
	if js == nil {
		return fmt.Errorf("JetStream not initialized")
	}

	// Create message payload
	message := map[string]interface{}{
		"type": "cover_generation_request",
		"data": CoverGenerationRequest{
			JobID:    jobID,
			PromptID: promptID,
		},
	}

	// Convert to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish to JetStream
	_, err = js.Publish(context.Background(), "cover.generate_request", payload)
	if err != nil {
		return err
	}

	log.Printf("Published cover letter generation request for job ID: %s", jobID)
	return nil
}

// JobStatusUpdateRequest represents a job status update request
type JobStatusUpdateRequest struct {
	JobID  int    `json:"job_id"`
	Status string `json:"status"`
}

// PublishJobStatusUpdateRequest publishes a job status update request message
func PublishJobStatusUpdateRequest(jobID int, status string) error {
	js := GetJetStream()
	if js == nil {
		return fmt.Errorf("JetStream not initialized")
	}

	// Create message payload
	message := map[string]interface{}{
		"type": "job_status_update_request",
		"data": JobStatusUpdateRequest{
			JobID:  jobID,
			Status: status,
		},
	}

	// Convert to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish to JetStream
	_, err = js.Publish(context.Background(), "jobs.status_update_request", payload)
	if err != nil {
		return err
	}

	log.Printf("Published job status update request for job ID: %d to status: %s", jobID, status)
	return nil
}

// SubscribeToJobsCreated subscribes to job creation messages
func SubscribeToJobsCreated(handler func(*nats.Msg)) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:    "cv-generator",
		Durable: "cv-generator",
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		// Convert jetstream.Msg to *nats.Msg for compatibility
		natsMsg := &nats.Msg{
			Subject: msg.Subject(),
			Data:    msg.Data(),
		}
		handler(natsMsg)
		msg.Ack()
	})
}

// SubscribeToJobCreationRequests subscribes to job creation request messages
func SubscribeToJobCreationRequests(handler func(*nats.Msg)) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "job-service",
		Durable:        "job-service",
		FilterSubjects: []string{"jobs.create_request"},
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		// Convert jetstream.Msg to *nats.Msg for compatibility
		natsMsg := &nats.Msg{
			Subject: msg.Subject(),
			Data:    msg.Data(),
		}
		handler(natsMsg)
		msg.Ack()
	})
}

// SubscribeToCVGenerated subscribes to CV generation completion messages
func SubscribeToCVGenerated(handler func(*nats.Msg)) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "pdf-generator",
		Durable:        "pdf-generator",
		FilterSubjects: []string{"cv.generated"},
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		// Convert jetstream.Msg to *nats.Msg for compatibility
		natsMsg := &nats.Msg{
			Subject: msg.Subject(),
			Data:    msg.Data(),
		}
		handler(natsMsg)
		msg.Ack()
	})
}

// SubscribeToJobsCreatedForCV subscribes to job creation messages for CV generation
func SubscribeToJobsCreatedForCV(handler func(*nats.Msg)) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "job-created-consumer",
		Durable:        "job-created-consumer",
		FilterSubjects: []string{"jobs.created"},
		AckWait:        5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		// Convert jetstream.Msg to *nats.Msg for compatibility
		natsMsg := &nats.Msg{
			Subject: msg.Subject(),
			Data:    msg.Data(),
		}
		handler(natsMsg)
		msg.Ack()
	})
}

// SubscribeToCVGenerationRequests subscribes to CV generation request messages
func SubscribeToCVGenerationRequests(handler func(*nats.Msg)) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "cv-generation-request-consumer",
		Durable:        "cv-generation-request-consumer",
		FilterSubjects: []string{"cv.generate_request"},
		AckWait:        5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		// Convert jetstream.Msg to *nats.Msg for compatibility
		natsMsg := &nats.Msg{
			Subject: msg.Subject(),
			Data:    msg.Data(),
		}
		handler(natsMsg)
		msg.Ack()
	})
}

// SubscribeToJobStatusUpdateRequests subscribes to job status update request messages
func SubscribeToJobStatusUpdateRequests(handler func(*nats.Msg)) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "job-status-update-consumer",
		Durable:        "job-status-update-consumer",
		FilterSubjects: []string{"jobs.status_update_request"},
		AckWait:        5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		// Convert jetstream.Msg to *nats.Msg for compatibility
		natsMsg := &nats.Msg{
			Subject: msg.Subject(),
			Data:    msg.Data(),
		}
		handler(natsMsg)
		msg.Ack()
	})
}

// PublishCVGeneratedMessage publishes a CV generated message to notify other services
func PublishCVGeneratedMessage(job models.Job) error {
	js := GetJetStream()
	if js == nil {
		return fmt.Errorf("JetStream not initialized")
	}

	// Create message payload
	message := map[string]interface{}{
		"type": "cv_generated",
		"data": job,
	}

	// Convert to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish to JetStream
	_, err = js.Publish(context.Background(), "jobs.cvgenerated", payload)
	if err != nil {
		return err
	}

	log.Printf("Published CV generated message for job %d", job.Id)
	return nil
}

// SubscribeToJobCreationRequestsGeneric subscribes to job creation request messages with generic handler
func SubscribeToJobCreationRequestsGeneric(handler MessageHandler) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "job-service",
		Durable:        "job-service",
		FilterSubjects: []string{"jobs.create_request"},
		AckWait:        5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg.Data()); err != nil {
			log.Printf("Error handling message: %v", err)
			msg.Nak()
		} else {
			msg.Ack()
		}
	})
}

// SubscribeToJobStatusUpdateRequestsGeneric subscribes to job status update request messages with generic handler
func SubscribeToJobStatusUpdateRequestsGeneric(handler MessageHandler) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "job-status-update-consumer",
		Durable:        "job-status-update-consumer",
		FilterSubjects: []string{"jobs.status_update_request"},
		AckWait:        5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg.Data()); err != nil {
			log.Printf("Error handling message: %v", err)
			msg.Nak()
		} else {
			msg.Ack()
		}
	})
}

// SubscribeToJobsCreatedForCVGeneric subscribes to job creation messages for CV generation with generic handler
func SubscribeToJobsCreatedForCVGeneric(handler MessageHandler) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "job-created-consumer",
		Durable:        "job-created-consumer",
		FilterSubjects: []string{"jobs.created"},
		AckWait:        5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg.Data()); err != nil {
			log.Printf("Error handling message: %v", err)
			msg.Nak()
		} else {
			msg.Ack()
		}
	})
}

// SubscribeToCVGenerationRequestsGeneric subscribes to CV generation request messages with generic handler
func SubscribeToCVGenerationRequestsGeneric(handler MessageHandler) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "cv-generation-request-consumer",
		Durable:        "cv-generation-request-consumer",
		FilterSubjects: []string{"cv.generate_request"},
		AckWait:        5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg.Data()); err != nil {
			log.Printf("Error handling message: %v", err)
			msg.Nak()
		} else {
			msg.Ack()
		}
	})
}

// SubscribeToCVGeneratedGeneric subscribes to CV generated messages with generic handler
func SubscribeToCVGeneratedGeneric(handler MessageHandler) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "cv-created-consumer",
		Durable:        "cv-created-consumer",
		FilterSubjects: []string{"jobs.cvgenerated"},
		AckWait:        5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg.Data()); err != nil {
			log.Printf("Error handling message: %v", err)
			msg.Nak()
		} else {
			msg.Ack()
		}
	})
}

// SubscribeToCVGeneratedForPDF subscribes to CV generated messages specifically for PDF generation
func SubscribeToCVGeneratedForPDF(handler MessageHandler) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "cv-pdf-generator-consumer",
		Durable:        "cv-pdf-generator-consumer",
		FilterSubjects: []string{"jobs.cvgenerated"},
		AckWait:        5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg.Data()); err != nil {
			log.Printf("Error handling message: %v", err)
			msg.Nak()
		} else {
			msg.Ack()
		}
	})
}

// SubscribeToCVGeneratedForScore subscribes to CV generated messages specifically for score generation
func SubscribeToCVGeneratedForScore(handler MessageHandler) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "cv-score-generator-consumer",
		Durable:        "cv-score-generator-consumer",
		FilterSubjects: []string{"jobs.cvgenerated"},
		AckWait:        5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg.Data()); err != nil {
			log.Printf("Error handling message: %v", err)
			msg.Nak()
		} else {
			msg.Ack()
		}
	})
}

// SubscribeToScoreGenerationRequestsGeneric subscribes to score generation request messages with generic handler
func SubscribeToScoreGenerationRequestsGeneric(handler MessageHandler) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "score-generation-request-consumer",
		Durable:        "score-generation-request-consumer",
		FilterSubjects: []string{"score.generate_request"},
		AckWait:        5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg.Data()); err != nil {
			log.Printf("Error handling message: %v", err)
			msg.Nak()
		} else {
			msg.Ack()
		}
	})
}

// SubscribeToJobsCreatedForCoverGeneric subscribes to job creation messages for cover letter generation with generic handler
func SubscribeToJobsCreatedForCoverGeneric(handler MessageHandler) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "job-created-cover-consumer",
		Durable:        "job-created-cover-consumer",
		FilterSubjects: []string{"jobs.created"},
		AckWait:        5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg.Data()); err != nil {
			log.Printf("Error handling message: %v", err)
			msg.Nak()
		} else {
			msg.Ack()
		}
	})
}

// SubscribeToCoverGenerationRequestsGeneric subscribes to cover letter generation request messages with generic handler
func SubscribeToCoverGenerationRequestsGeneric(handler MessageHandler) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "cover-generation-request-consumer",
		Durable:        "cover-generation-request-consumer",
		FilterSubjects: []string{"cover.generate_request"},
		AckWait:        5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg.Data()); err != nil {
			log.Printf("Error handling message: %v", err)
			msg.Nak()
		} else {
			msg.Ack()
		}
	})
}

// PublishCoverGeneratedMessage publishes a cover letter generated message to notify other services
func PublishCoverGeneratedMessage(job models.Job) error {
	js := GetJetStream()
	if js == nil {
		return fmt.Errorf("JetStream not initialized")
	}

	// Create message payload
	message := map[string]interface{}{
		"type": "cover_generated",
		"data": job,
	}

	// Convert to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish to JetStream
	_, err = js.Publish(context.Background(), "cover.generated", payload)
	if err != nil {
		return err
	}

	log.Printf("Published cover letter generated message for job %d", job.Id)
	return nil
}

// SubscribeToCoverGeneratedGeneric subscribes to cover letter generated messages with generic handler
func SubscribeToCoverGeneratedGeneric(handler MessageHandler) (jetstream.ConsumeContext, error) {
	js := GetJetStream()
	if js == nil {
		return nil, fmt.Errorf("JetStream not initialized")
	}

	consumer, err := js.CreateOrUpdateConsumer(context.Background(), "JOBS", jetstream.ConsumerConfig{
		Name:           "cover-generated-consumer",
		Durable:        "cover-generated-consumer",
		FilterSubjects: []string{"cover.generated"},
		AckWait:        5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg.Data()); err != nil {
			log.Printf("Error handling message: %v", err)
			msg.Nak()
		} else {
			msg.Ack()
		}
	})
}