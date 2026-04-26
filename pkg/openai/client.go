// Package openai provides a Bonsai Brain ModelClient that talks to any
// OpenAI-compatible endpoint: llama.cpp server, Ollama (with OpenAI flag),
// vLLM, TGI, Groq, OpenRouter, etc.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/donn/bonsai-brain/pkg/engine"
)

// Client is an OpenAI-compatible model backend.
type Client struct {
	BaseURL      string
	APIKey       string
	Model        string
	ExtraHeaders map[string]string
	client       *http.Client
}

// NewClient creates an OpenAI-compatible client.
func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// Stream implements engine.ModelClient.
func (c *Client) Stream(ctx context.Context, messages []engine.Message, tools []engine.ToolSchema) (*engine.Response, error) {
	type toolCall struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
	type msg struct {
		Role       string     `json:"role"`
		Content    string     `json:"content"`
		ToolCalls  []toolCall `json:"tool_calls,omitempty"`
		ToolCallID string     `json:"tool_call_id,omitempty"`
	}
	type toolDef struct {
		Type     string `json:"type"`
		Function struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			Parameters  map[string]any `json:"parameters"`
		} `json:"function"`
	}

	var msgs []msg
	for _, m := range messages {
		out := msg{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
		for _, tc := range m.ToolCalls {
			args, _ := json.Marshal(tc.Args)
			out.ToolCalls = append(out.ToolCalls, toolCall{
				ID:   tc.ID,
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{Name: tc.Name, Arguments: string(args)},
			})
		}
		msgs = append(msgs, out)
	}

	var toolList []toolDef
	for _, t := range tools {
		toolList = append(toolList, toolDef{
			Type: "function",
			Function: struct {
				Name        string         `json:"name"`
				Description string         `json:"description"`
				Parameters  map[string]any `json:"parameters"`
			}{Name: t.Name, Description: t.Description, Parameters: t.Parameters},
		})
	}

	body := map[string]any{
		"model":       c.Model,
		"messages":    msgs,
		"tools":       toolList,
		"tool_choice": "auto",
		"max_tokens":  512,
		"temperature": 0.3,
	}
	if len(toolList) == 0 {
		delete(body, "tools")
		delete(body, "tool_choice")
	}

	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.ExtraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("openai error %d: %v", resp.StatusCode, errBody)
	}

	var payload struct {
		Choices []struct {
			Message struct {
				Role       string     `json:"role"`
				Content    string     `json:"content"`
				ToolCalls  []toolCall `json:"tool_calls"`
				Refusal    string     `json:"refusal"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("openai decode error: %w", err)
	}
	if len(payload.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	choice := payload.Choices[0]
	out := &engine.Response{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
	}
	for _, tc := range choice.Message.ToolCalls {
		var args map[string]any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		out.ToolCalls = append(out.ToolCalls, engine.ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: args,
		})
	}
	return out, nil
}
