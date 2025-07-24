package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// AIProvider represents different AI providers
type AIProvider string

const (
	ProviderGemini AIProvider = "gemini"
	ProviderOllama AIProvider = "ollama"
)

// AIClient interface for different AI providers
type AIClient interface {
	Generate(prompt string) (string, error)
}

// GeminiClient implements AIClient for Google Gemini
type GeminiClient struct {
	APIKey string
	Model  string
}

// OllamaClient implements AIClient for Ollama (placeholder for future implementation)
type OllamaClient struct {
	BaseURL string
	Model   string
}

// NewClient creates a new AI client based on the provider
func NewClient(provider AIProvider) (AIClient, error) {
	switch provider {
	case ProviderGemini:
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY not set")
		}
		return &GeminiClient{
			APIKey: apiKey,
			Model:  "gemini-2.5-pro",
		}, nil
	case ProviderOllama:
		baseURL := os.Getenv("OLLAMA_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:11434" // Default Ollama URL
		}
		model := os.Getenv("OLLAMA_MODEL")
		if model == "" {
			model = "llama2" // Default model
		}
		return &OllamaClient{
			BaseURL: baseURL,
			Model:   model,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", provider)
	}
}

// Generate implements AIClient for GeminiClient
func (g *GeminiClient) Generate(prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.Model, g.APIKey)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

// Generate implements AIClient for OllamaClient (placeholder implementation)
func (o *OllamaClient) Generate(prompt string) (string, error) {
	// TODO: Implement Ollama integration
	return "", fmt.Errorf("Ollama integration not yet implemented")
}

// DefaultClient creates a default AI client (Gemini for now)
func DefaultClient() (AIClient, error) {
	return NewClient(ProviderGemini)
}