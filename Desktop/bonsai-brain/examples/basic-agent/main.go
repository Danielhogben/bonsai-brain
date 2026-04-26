package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/donn/bonsai-brain/pkg/agent"
	"github.com/donn/bonsai-brain/pkg/engine"
	"github.com/donn/bonsai-brain/pkg/guardrail"
	"github.com/donn/bonsai-brain/pkg/middleware"
)

// ---------------------------------------------------------------------------
// MockModel simulates an LLM for demonstration purposes.
// In production, replace this with a real client (OpenAI, Groq, Ollama, etc.).
// ---------------------------------------------------------------------------
type MockModel struct{}

func (m *MockModel) Stream(ctx context.Context, messages []engine.Message, tools []engine.ToolSchema) (*engine.Response, error) {
	// Find the last user message.
	var lastUser string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUser = messages[i].Content
			break
		}
	}

	// If the user asked for the weather, simulate a tool call.
	if strings.Contains(strings.ToLower(lastUser), "weather") {
		return &engine.Response{
			Content: "",
			ToolCalls: []engine.ToolCall{
				{ID: "call_1", Name: "get_weather", Args: map[string]any{"city": "Tokyo"}},
			},
			FinishReason: "tool_calls",
		}, nil
	}

	return &engine.Response{
		Content:      fmt.Sprintf("MockModel says: I received %d messages.", len(messages)),
		FinishReason: "stop",
	}, nil
}

func main() {
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// 1. Engine
	// -------------------------------------------------------------------------
	eng := engine.NewQueryEngine(&MockModel{})

	// Register a weather tool.
	eng.RegisterTool(
		engine.ToolSchema{
			Name:        "get_weather",
			Description: "Get the current weather for a city",
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
			return fmt.Sprintf("Weather in %s: 22°C, sunny", city), nil
		},
	)

	// Require user approval for the weather tool.
	eng.Permission = func(call engine.ToolCall) engine.PermissionDecision {
		if call.Name == "get_weather" {
			return engine.PermissionAskUser
		}
		return engine.PermissionAllow
	}

	// Auto-approve for this demo.
	eng.AskUser = func(call engine.ToolCall) bool {
		fmt.Printf("[APPROVAL] Allow tool %q? yes\n", call.Name)
		return true
	}

	// -------------------------------------------------------------------------
	// 2. Agent with guardrails and middleware
	// -------------------------------------------------------------------------
	cfg := agent.DefaultConfig("demo-agent")
	cfg.SystemPrompt = "You are a concise assistant."
	ag := agent.New(cfg, eng)

	// Input guardrail: block inputs containing "password".
	ag.InGuardrails.Add(guardrail.BlockedKeywords("password", "secret"))

	// Input guardrail: max 500 characters.
	ag.InGuardrails.Add(guardrail.MaxInputLength(500))

	// Output middleware: truncate to 200 characters.
	ag.OutMiddleware.Add(middleware.TruncateOutput(200))

	// -------------------------------------------------------------------------
	// 3. Run
	// -------------------------------------------------------------------------
	reply, err := ag.GenerateText(ctx, "What's the weather like today?")
	if err != nil {
		log.Fatalf("agent error: %v", err)
	}
	fmt.Printf("\nFinal reply:\n%s\n", reply)
}
