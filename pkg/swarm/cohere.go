package swarm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/donn/bonsai-brain/pkg/engine"
)

// CohereClient adapts Cohere's native /v1/chat API to engine.ModelClient.
type CohereClient struct {
	BaseURL string
	APIKey  string
	Model   string
	client  *http.Client
}

// NewCohereClient creates a Cohere API client.
func NewCohereClient(baseURL, apiKey, model string) *CohereClient {
	if baseURL == "" {
		baseURL = "https://api.cohere.com"
	}
	return &CohereClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// Stream implements engine.ModelClient using Cohere's /v1/chat endpoint.
func (c *CohereClient) Stream(ctx context.Context, messages []engine.Message, _ []engine.ToolSchema) (*engine.Response, error) {
	// Cohere uses "message" for the latest user message and "chat_history"
	// for prior turns. System messages go in "preamble".
	var preamble string
	var history []cohereMessage
	var lastUser string

	for _, m := range messages {
		switch m.Role {
		case "system":
			preamble = m.Content
		case "user":
			if lastUser != "" {
				// Push previous user message into history.
				history = append(history, cohereMessage{Role: "USER", Message: lastUser})
			}
			lastUser = m.Content
		case "assistant":
			if lastUser != "" {
				history = append(history, cohereMessage{Role: "USER", Message: lastUser})
				lastUser = ""
			}
			history = append(history, cohereMessage{Role: "CHATBOT", Message: m.Content})
		}
	}

	body := map[string]any{
		"message":      lastUser,
		"model":        c.Model,
		"chat_history": history,
		"temperature":  0.3,
	}
	if preamble != "" {
		body["preamble"] = preamble
	}
	if lastUser == "" {
		lastUser = "Hi"
		body["message"] = lastUser
	}

	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cohere http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("cohere error %d: %v", resp.StatusCode, errBody)
	}

	var payload struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("cohere decode error: %w", err)
	}

	return &engine.Response{
		Content:      payload.Text,
		FinishReason: "stop",
	}, nil
}

type cohereMessage struct {
	Role    string `json:"role"`
	Message string `json:"message"`
}
