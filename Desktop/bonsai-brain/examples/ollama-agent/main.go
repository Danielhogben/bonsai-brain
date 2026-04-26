package main

import (
	"context"
	"fmt"
	"os"

	"github.com/donn/bonsai-brain/pkg/agent"
	"github.com/donn/bonsai-brain/pkg/engine"
	"github.com/donn/bonsai-brain/pkg/guardrail"
	"github.com/donn/bonsai-brain/pkg/ollama"
)

func main() {
	ctx := context.Background()

	fmt.Println("╔═══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║     🌳  BONSAI BRAIN + OLLAMA — Local Agent Demo                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Backend: Ollama @ http://127.0.0.1:11435")
	fmt.Println("Model:   qwen2.5-tiny (0.5B params, ~400 MB)")
	fmt.Println("Device:  CPU-only, no CUDA required")
	fmt.Println()

	// 1. Wire Bonsai Brain to Ollama
	client := ollama.NewClient("http://127.0.0.1:11435", "qwen2.5-tiny")
	eng := engine.NewQueryEngine(client)

	// 2. Register tiny tools
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
			Name:        "get_os",
			Description: "Return the operating system name",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		},
		func(_ context.Context, _ map[string]any) (string, error) {
			return os.Getenv("GOOS"), nil
		},
	)

	// 3. Agent with guardrails
	cfg := agent.DefaultConfig("ollama-demo")
	cfg.SystemPrompt = "You are a concise assistant running on tiny hardware. Use tools when asked about the system. Keep answers to one sentence."
	ag := agent.New(cfg, eng)
	ag.InGuardrails.Add(guardrail.MaxInputLength(500))

	// 4. Run demo queries
	// Note: 0.5B models struggle with structured tool output. Tool calling
	// works reliably with 1B+ models (qwen2.5:1.5b, llama3.2:3b, etc.)
	queries := []struct {
		q   string
		note string
	}{
		{"What is the hostname of this machine?", "(may use get_hostname tool — 0.5b models are inconsistent)"},
		{"What operating system am I running?", "(may use get_os tool)"},
		{"Tell me a fun fact about bonsai trees.", "(pure text — no tools needed)"},
	}

	for _, item := range queries {
		fmt.Printf("\n┌─ User: %s %s\n", item.q, item.note)
		reply, err := ag.GenerateText(ctx, item.q)
		if err != nil {
			fmt.Printf("└─ Error: %v\n", err)
			continue
		}
		fmt.Printf("└─ Bonsai Brain: %s\n", reply)
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════════════")
	fmt.Println("  ✅ Ollama integration demo complete")
	fmt.Println("═══════════════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("Tip: For reliable tool calling, use a 1B–3B model:")
	fmt.Println("  ollama pull qwen2.5:1.5b")
	fmt.Println("  ollama pull llama3.2:3b")
}
