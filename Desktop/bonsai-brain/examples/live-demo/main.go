package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/donn/bonsai-brain/pkg/agent"
	"github.com/donn/bonsai-brain/pkg/engine"
	"github.com/donn/bonsai-brain/pkg/guardrail"
	"github.com/donn/bonsai-brain/pkg/middleware"
)

// ---------------------------------------------------------------------------
// OpenAI-compatible client talking to the local proxy
// ---------------------------------------------------------------------------
type ProxyClient struct {
	BaseURL string
	APIKey  string
	Model   string
	client  *http.Client
}

func NewProxyClient(baseURL, apiKey, model string) *ProxyClient {
	return &ProxyClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *ProxyClient) Stream(ctx context.Context, messages []engine.Message, tools []engine.ToolSchema) (*engine.Response, error) {
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
		"max_tokens":  512,
		"temperature": 0.3,
	}
	if len(toolList) == 0 {
		delete(body, "tools")
	}

	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("api error %d: %v", resp.StatusCode, errBody)
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
		return nil, fmt.Errorf("decode error: %w", err)
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

// ---------------------------------------------------------------------------
// Tools
// ---------------------------------------------------------------------------

func hostnameTool() (engine.ToolSchema, engine.ToolExecutor) {
	return engine.ToolSchema{
			Name:        "get_hostname",
			Description: "Return the system hostname",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		func(_ context.Context, _ map[string]any) (string, error) {
			h, _ := os.Hostname()
			return h, nil
		}
}

func timeTool() (engine.ToolSchema, engine.ToolExecutor) {
	return engine.ToolSchema{
			Name:        "get_current_time",
			Description: "Return the current date and time",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		func(_ context.Context, _ map[string]any) (string, error) {
			return time.Now().Format("Mon Jan 2 15:04:05 MST 2006"), nil
		}
}

func weatherTool() (engine.ToolSchema, engine.ToolExecutor) {
	return engine.ToolSchema{
			Name:        "get_weather",
			Description: "Get a fictional weather report for a city",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]any{"type": "string", "description": "City name"},
				},
				"required": []string{"city"},
			},
		},
		func(_ context.Context, args map[string]any) (string, error) {
			city := args["city"].(string)
			conditions := []string{"sunny", "cloudy", "light rain", "clear skies", "windy"}
			temp := 18 + len(city)%15
			cond := conditions[len(city)%len(conditions)]
			return fmt.Sprintf("%s: %d°C, %s", city, temp, cond), nil
		}
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	ctx := context.Background()

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║           🌳  BONSAI BRAIN v3 — LIVE DEMO                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 1. Engine wired to the local proxy
	client := NewProxyClient("http://127.0.0.1:8787/v1", "dummy", "mistralai/mistral-7b-instruct:free")
	eng := engine.NewQueryEngine(client)

	// 2. Register tools
	schema, exec := hostnameTool()
	eng.RegisterTool(schema, exec)
	schema, exec = timeTool()
	eng.RegisterTool(schema, exec)
	schema, exec = weatherTool()
	eng.RegisterTool(schema, exec)

	// 3. Permission pipeline — ask user before weather & hostname
	eng.Permission = func(call engine.ToolCall) engine.PermissionDecision {
		if call.Name == "get_weather" || call.Name == "get_hostname" {
			return engine.PermissionAskUser
		}
		return engine.PermissionAllow
	}
	eng.AskUser = func(call engine.ToolCall) bool {
		fmt.Printf("\n[🔒 PERMISSION] Allow tool %q? (y/n): ", call.Name)
		reader := bufio.NewReader(os.Stdin)
		txt, _ := reader.ReadString('\n')
		return strings.TrimSpace(strings.ToLower(txt)) == "y"
	}

	// 4. Agent with guardrails & middleware
	cfg := agent.DefaultConfig("live-demo")
	cfg.SystemPrompt = "You are Bonsai Brain, a concise assistant. Use tools when needed. Keep answers short."
	ag := agent.New(cfg, eng)

	ag.InGuardrails.Add(guardrail.MaxInputLength(1000))
	ag.InGuardrails.Add(guardrail.BlockedKeywords("password", "secret", "token"))
	ag.OutMiddleware.Add(middleware.TruncateOutput(800))

	// 5. Interactive loop
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Ask me anything. Try:")
	fmt.Println("  • 'What time is it?'")
	fmt.Println("  • 'What's the weather in Tokyo?'")
	fmt.Println("  • 'What machine am I running on?'")
	fmt.Println("  • 'password123' (should be blocked)")
	fmt.Println("  • 'quit' to exit")
	fmt.Println()

	for {
		fmt.Print("You: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if strings.ToLower(input) == "quit" {
			fmt.Println("\nBonsai Brain: Goodbye! 🌳")
			break
		}

		reply, err := ag.GenerateText(ctx, input)
		if err != nil {
			fmt.Printf("\nBonsai Brain: [ERROR] %v\n\n", err)
			continue
		}
		fmt.Printf("\nBonsai Brain: %s\n\n", reply)
	}
}
