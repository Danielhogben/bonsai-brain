package main

import (
	"context"
	"fmt"
	"os"

	"github.com/donn/bonsai-brain/pkg/agent"
	"github.com/donn/bonsai-brain/pkg/engine"
	"github.com/donn/bonsai-brain/pkg/guardrail"
	"github.com/donn/bonsai-brain/pkg/middleware"
	"github.com/donn/bonsai-brain/pkg/openai"
)

// ProxyClient reuses the OpenAI-compatible client from the live-demo example.
// llama-server on port 11434 exposes an OpenAI-compatible API.
type ProxyClient struct {
	BaseURL string
	APIKey  string
	Model   string
}

func main() {
	ctx := context.Background()

	fmt.Println("╔═══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║     🌳  BONSAI BRAIN + PRISMML BONSAI 1.7B (1-bit)                   ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Backend:  llama-server @ http://127.0.0.1:11434")
	fmt.Println("Model:    Bonsai-1.7B-Q1_0 (1-bit, 237 MB, 1.7B params)")
	fmt.Println("Format:   Q1_0 — ultra-compressed GGUF")
	fmt.Println("Memory:   ~350 MB RAM at 2K context")
	fmt.Println()

	// Wire Bonsai Brain to the local llama-server via OpenAI-compatible API.
	eng := engine.NewQueryEngine(openai.NewClient(
		"http://127.0.0.1:11434/v1",
		"dummy",
		"bonsai-1.7b",
	))

	// Register simple system tools
	eng.RegisterTool(
		engine.ToolSchema{
			Name:        "get_hostname",
			Description: "Return the system hostname",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		},
		func(_ context.Context, _ map[string]any) (string, error) {
			return os.Hostname()
		},
	)

	eng.RegisterTool(
		engine.ToolSchema{
			Name:        "get_current_time",
			Description: "Return the current date and time",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		},
		func(_ context.Context, _ map[string]any) (string, error) {
			return fmt.Sprintf("It is currently %s", "[time tool result]"), nil
		},
	)

	// 0.5B–1.7B models struggle with structured tool output.
	// For reliable tool calling, use 3B+ models.
	eng.Permission = func(_ engine.ToolCall) engine.PermissionDecision {
		return engine.PermissionAllow
	}

	cfg := agent.DefaultConfig("prism-bonsai")
	cfg.SystemPrompt = "You are a helpful assistant running the PrismML Bonsai 1.7B 1-bit model. Be concise."
	ag := agent.New(cfg, eng)
	ag.InGuardrails.Add(guardrail.MaxInputLength(500))
	ag.OutMiddleware.Add(middleware.TruncateOutput(300))

	queries := []string{
		"What is a bonsai tree?",
		"Explain 1-bit quantization in one sentence.",
		"What is the hostname of this machine?",
	}

	for _, q := range queries {
		fmt.Printf("\n┌─ User: %s\n", q)
		reply, err := ag.GenerateText(ctx, q)
		if err != nil {
			fmt.Printf("└─ Error: %v\n", err)
			continue
		}
		fmt.Printf("└─ Bonsai Brain: %s\n", reply)
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════════════")
	fmt.Println("  ✅ PrismML Bonsai integration demo complete")
	fmt.Println("═══════════════════════════════════════════════════════════════════════")
}

