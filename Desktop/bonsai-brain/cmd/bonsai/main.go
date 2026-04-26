// Bonsai Brain CLI — single-binary agent runner.
//
// Usage:
//
//	bonsai run --config agent.yaml    # Run agent from config file
//	bonsai chat                       # Interactive REPL
//	bonsai version                    # Show version
//
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/donn/bonsai-brain/pkg/agent"
	"github.com/donn/bonsai-brain/pkg/engine"
	"github.com/donn/bonsai-brain/pkg/guardrail"
	"github.com/donn/bonsai-brain/pkg/middleware"
)

const version = "0.3.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "chat":
		chatCmd(os.Args[2:])
	case "version":
		fmt.Println("bonsai", version)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: bonsai <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  run      Run an agent from a YAML config file")
	fmt.Println("  chat     Start an interactive chat session")
	fmt.Println("  version  Show version")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  bonsai run --config agent.yaml")
	fmt.Println("  bonsai chat --model qwen/qwen3-32b")
}

// ---------------------------------------------------------------------------
// run
// ---------------------------------------------------------------------------

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", "agent.yaml", "Path to agent config file")
	_ = fs.Parse(args)

	fmt.Printf("🌳 Bonsai Brain v%s — run mode\n", version)
	fmt.Printf("Loading config: %s\n", *configPath)
	fmt.Println("(YAML config loader coming in v0.4 — using built-in demo for now)")
	fmt.Println()

	// Demo run with mock model until YAML loader is implemented.
	ctx := context.Background()
	eng := engine.NewQueryEngine(&MockModel{})
	eng.RegisterTool(
		engine.ToolSchema{Name: "echo", Description: "Echo input", Parameters: map[string]any{"type": "object", "properties": map[string]any{"text": map[string]any{"type": "string"}}, "required": []string{"text"}}},
		func(_ context.Context, args map[string]any) (string, error) { return args["text"].(string), nil },
	)

	cfg := agent.DefaultConfig("cli-demo")
	cfg.SystemPrompt = "You are Bonsai Brain CLI. Be concise."
	ag := agent.New(cfg, eng)
	ag.InGuardrails.Add(guardrail.MaxInputLength(500))
	ag.OutMiddleware.Add(middleware.TruncateOutput(300))

	reply, err := ag.GenerateText(ctx, "Echo 'Bonsai Brain CLI is running!'")
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	fmt.Println(reply)
}

// ---------------------------------------------------------------------------
// chat
// ---------------------------------------------------------------------------

func chatCmd(args []string) {
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	modelFlag := fs.String("model", "mistralai/mistral-7b-instruct:free", "Model to use")
	_ = fs.Parse(args)

	fmt.Printf("🌳 Bonsai Brain v%s — chat mode\n", version)
	fmt.Printf("Model: %s\n", *modelFlag)
	fmt.Println("(Connect a real model client in your integration. Using mock for demo.)")
	fmt.Println()
	fmt.Println("Type 'quit' to exit, 'clear' to reset history.")
	fmt.Println()

	ctx := context.Background()
	eng := engine.NewQueryEngine(&MockModel{})
	cfg := agent.DefaultConfig("chat")
	cfg.SystemPrompt = "You are a helpful assistant. Keep responses brief."
	ag := agent.New(cfg, eng)
	ag.InGuardrails.Add(guardrail.BlockedKeywords("password", "secret"))

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("You: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if strings.ToLower(input) == "quit" {
			fmt.Println("Bonsai Brain: Goodbye! 🌳")
			break
		}
		if strings.ToLower(input) == "clear" {
			ag.Ctx.History = nil
			fmt.Println("Bonsai Brain: History cleared.")
			continue
		}

		reply, err := ag.GenerateText(ctx, input)
		if err != nil {
			fmt.Printf("Bonsai Brain: [ERROR] %v\n\n", err)
			continue
		}
		fmt.Printf("Bonsai Brain: %s\n\n", reply)
	}
}

// ---------------------------------------------------------------------------
// MockModel
// ---------------------------------------------------------------------------

type MockModel struct{}

func (m *MockModel) Stream(_ context.Context, messages []engine.Message, _ []engine.ToolSchema) (*engine.Response, error) {
	var lastUser string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUser = messages[i].Content
			break
		}
	}
	if strings.Contains(lastUser, "echo") {
		parts := strings.SplitN(lastUser, "'", 3)
		if len(parts) >= 2 {
			return &engine.Response{Content: parts[1], FinishReason: "stop"}, nil
		}
	}
	return &engine.Response{Content: "MockModel received: " + lastUser, FinishReason: "stop"}, nil
}
