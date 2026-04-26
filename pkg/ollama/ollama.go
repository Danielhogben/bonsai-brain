// Package ollama provides a Bonsai Brain ModelClient that talks directly to
// an Ollama server. It works with any model Ollama can load — no OpenAI
// compatibility layer required.
//
// Tool calling is implemented via prompt engineering: tool schemas are
// injected into the system prompt, and the model's response is parsed for
// tool call patterns. This lets tiny local models (0.5B–3B) participate in
// agent loops without native function-calling support.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/donn/bonsai-brain/pkg/engine"
)

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client is an Ollama-native model backend.
type Client struct {
	BaseURL string
	Model   string
	client  *http.Client
}

// NewClient creates an Ollama client.
func NewClient(baseURL, model string) *Client {
	return &Client{
		BaseURL: baseURL,
		Model:   model,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// Stream implements engine.ModelClient.
func (c *Client) Stream(ctx context.Context, messages []engine.Message, tools []engine.ToolSchema) (*engine.Response, error) {
	// Convert engine messages to Ollama format.
	ollamaMsgs := c.toOllamaMessages(messages, tools)

	body := map[string]any{
		"model":    c.Model,
		"messages": ollamaMsgs,
		"stream":   false,
		"options": map[string]any{
			"temperature": 0.3,
			"num_ctx":     2048,
		},
	}

	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/chat", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("ollama error %d: %v", resp.StatusCode, errBody)
	}

	var payload struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		DoneReason string `json:"done_reason"`
		Done       bool   `json:"done"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("ollama decode error: %w", err)
	}

	content := strings.TrimSpace(payload.Message.Content)

	// Try to parse tool calls from the content.
	toolCalls := c.parseToolCalls(content, tools)
	if len(toolCalls) > 0 {
		return &engine.Response{
			Content:      "",
			ToolCalls:    toolCalls,
			FinishReason: "tool_calls",
		}, nil
	}

	return &engine.Response{
		Content:      content,
		FinishReason: "stop",
	}, nil
}

// ---------------------------------------------------------------------------
// Message conversion
// ---------------------------------------------------------------------------

func (c *Client) toOllamaMessages(messages []engine.Message, tools []engine.ToolSchema) []map[string]string {
	var out []map[string]string

	// Build tool instructions if tools are present.
	var toolInstructions string
	if len(tools) > 0 {
		var parts []string
		parts = append(parts, "TOOLS:")
		for _, t := range tools {
			parts = append(parts, fmt.Sprintf("- %s: %s", t.Name, t.Description))
		}
		parts = append(parts, "")
		parts = append(parts, "To use a tool, reply EXACTLY with:")
		parts = append(parts, "TOOL: tool_name({\"arg\":\"value\"})")
		parts = append(parts, "Otherwise reply normally.")
		toolInstructions = strings.Join(parts, "\n")
	}

	for _, m := range messages {
		role := m.Role
		content := m.Content

		// Inject tool instructions into the system prompt.
		if role == "system" && toolInstructions != "" {
			content = content + "\n\n" + toolInstructions
		}

		out = append(out, map[string]string{
			"role":    role,
			"content": content,
		})
	}

	return out
}

// ---------------------------------------------------------------------------
// Tool call parsing
// ---------------------------------------------------------------------------

// parseToolCalls tries multiple patterns because tiny models are inconsistent.
func (c *Client) parseToolCalls(content string, tools []engine.ToolSchema) []engine.ToolCall {
	// Build valid tool name set.
	valid := make(map[string]bool, len(tools))
	for _, t := range tools {
		valid[t.Name] = true
	}

	var calls []engine.ToolCall

	// Pattern 1: TOOL: name({...})
	re1 := regexp.MustCompile(`(?i)TOOL:\s*(\w+)\s*\((.*?)\)\s*$`)
	for _, m := range re1.FindAllStringSubmatch(content, -1) {
		if call := c.tryParseCall(m[1], m[2], valid, len(calls)); call != nil {
			calls = append(calls, *call)
		}
	}

	// Pattern 2: name({...}) anywhere in text
	re2 := regexp.MustCompile(`(?i)\b(\w+)\s*\(\s*(\{.*?\})\s*\)`)
	for _, m := range re2.FindAllStringSubmatch(content, -1) {
		if call := c.tryParseCall(m[1], m[2], valid, len(calls)); call != nil {
			calls = append(calls, *call)
		}
	}

	return calls
}

func (c *Client) tryParseCall(name, argsJSON string, valid map[string]bool, idx int) *engine.ToolCall {
	if !valid[name] {
		return nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		_ = json.Unmarshal([]byte("{"+argsJSON+"}"), &args)
	}
	return &engine.ToolCall{
		ID:   fmt.Sprintf("call_%d", idx+1),
		Name: name,
		Args: args,
	}
}
