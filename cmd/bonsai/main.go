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
	"sort"
	"strings"
	"time"

	"github.com/donn/bonsai-brain/pkg/agent"
	"github.com/donn/bonsai-brain/pkg/engine"
	"github.com/donn/bonsai-brain/pkg/guardrail"
	"github.com/donn/bonsai-brain/pkg/middleware"
	"github.com/donn/bonsai-brain/pkg/swarm"
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
	case "swarm":
		swarmCmd(os.Args[2:])
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
	fmt.Println("  swarm    Launch the full cloud swarm stack")
	fmt.Println("  version  Show version")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  bonsai run --config agent.yaml")
	fmt.Println("  bonsai chat --model qwen/qwen3-32b")
	fmt.Println("  bonsai swarm")
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
// swarm
// ---------------------------------------------------------------------------

func swarmCmd(args []string) {
	fs := flag.NewFlagSet("swarm", flag.ExitOnError)
	prompt := fs.String("prompt", "Explain the concept of 'swarm intelligence' in 2 sentences. Be concise.", "Task prompt to send to all agents")
	_ = fs.Parse(args)

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("  🌳 BONSAI BRAIN — FULL CLOUD SWARM STACK")
	fmt.Println("  Maxing out every free API key we have...")
	fmt.Println("═══════════════════════════════════════════════════════════════")

	ctx := context.Background()

	// Try to load swarm.yaml config first
	var configs []swarm.ProviderConfig
	if cfg, err := swarm.LoadSwarmYAML("swarm.yaml"); err == nil {
		fmt.Println("📄 Loaded swarm.yaml")
		configs = cfg.ToProviderConfigs()
	} else {
		fmt.Printf("⚠️  swarm.yaml not loaded (%v), using defaults\n", err)
		configs = swarm.DefaultProviderConfigs()
	}
	active := swarm.ActiveProviders(configs)

	fmt.Printf("\n📡 Providers found:\n")
	for _, c := range active {
		keyStatus := "✅"
		if c.APIKey == "" && c.Type != swarm.ProviderOllama && c.Type != swarm.ProviderLocal {
			keyStatus = "⚠️  no key"
		}
		fmt.Printf("   %s %s — %d models (rate: %d/min, timeout: %ds)\n",
			keyStatus, c.Type, len(c.Models), c.RateLimit, c.TimeoutSec)
	}
	if len(active) == 0 {
		fmt.Println("❌ No active providers. Set at least one API key in ~/.hermes/.env")
		os.Exit(1)
	}

	registry := swarm.NewProviderRegistry(active)
	swarmInst := swarm.NewSwarm(registry)

	fmt.Println("\n🤖 Spawning agents...")
	spawned, err := swarmInst.SpawnAll()
	if err != nil {
		fmt.Printf("❌ Spawn error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Spawned %d agents\n", len(spawned))

	task := swarm.Task{
		ID:      "swarm-cli-1",
		Prompt:  *prompt,
		System:  "You are a helpful assistant. Be concise.",
		MaxIter: 1,
	}

	fmt.Printf("\n📨 Dispatching task to all agents:\n")
	fmt.Printf("   \"%s\"\n", task.Prompt)

	dispatchCtx, dispatchCancel := context.WithTimeout(ctx, 180*time.Second)
	defer dispatchCancel()

	start := time.Now()
	results := swarmInst.Distribute(dispatchCtx, task)
	elapsed := time.Since(start)

	fmt.Println("\n═══════════════════════════════════════════════════════════════")
	fmt.Println("  RESULTS")
	fmt.Println("═══════════════════════════════════════════════════════════════")

	sort.Slice(results, func(i, j int) bool {
		if results[i].Error != nil && results[j].Error == nil {
			return false
		}
		if results[i].Error == nil && results[j].Error != nil {
			return true
		}
		return results[i].Latency < results[j].Latency
	})

	fmt.Printf("\n%-40s %-12s %-10s %s\n", "Agent", "Latency", "Status", "Output (truncated)")
	fmt.Println(strings.Repeat("─", 120))
	for _, r := range results {
		status := "✅"
		out := strings.ReplaceAll(r.Output, "\n", " ")
		if len(out) > 70 {
			out = out[:67] + "..."
		}
		if r.Error != nil {
			status = "❌"
			out = r.Error.Error()
			if len(out) > 70 {
				out = out[:67] + "..."
			}
		} else if out == "" {
			status = "⚠️"
			out = "(empty response)"
		}
		fmt.Printf("%-40s %-12v %-10s %s\n", r.AgentID, r.Latency, status, out)
	}

	fmt.Println("\n═══════════════════════════════════════════════════════════════")
	fmt.Println("  AGGREGATION")
	fmt.Println("═══════════════════════════════════════════════════════════════")

	if first, err := swarm.FirstWinner(results); err == nil {
		fmt.Printf("\n🏁 FIRST WINNER\n   Agent: %s\n   Model: %s\n   Latency: %v\n   Output: %s\n",
			first.Winner.AgentID, first.Winner.Model, first.Winner.Latency, first.Winner.Output)
	}
	if fastest, err := swarm.FastestWinner(results); err == nil {
		fmt.Printf("\n⚡ FASTEST WINNER\n   Agent: %s\n   Model: %s\n   Latency: %v\n   Output: %s\n",
			fastest.Winner.AgentID, fastest.Winner.Model, fastest.Winner.Latency, fastest.Winner.Output)
	}
	if consensus, err := swarm.ConsensusWinner(results); err == nil {
		fmt.Printf("\n🗳️  CONSENSUS WINNER\n   Strategy: %s\n   Output: %s\n",
			consensus.Description, consensus.Winner.Output)
	}

	fmt.Println("\n═══════════════════════════════════════════════════════════════")
	fmt.Println("  SUMMARY")
	fmt.Println("═══════════════════════════════════════════════════════════════")

	success := 0
	fail := 0
	var totalLat time.Duration
	for _, r := range results {
		if r.Error == nil && r.Output != "" {
			success++
			totalLat += r.Latency
		} else {
			fail++
		}
	}
	fmt.Printf("   Total agents:    %d\n", len(results))
	fmt.Printf("   Success:         %d\n", success)
	fmt.Printf("   Failed:          %d\n", fail)
	fmt.Printf("   Wall-clock time: %v\n", elapsed)
	if success > 0 {
		fmt.Printf("   Avg latency:     %v\n", totalLat/time.Duration(success))
	}

	fmt.Println("\n═══════════════════════════════════════════════════════════════")
	fmt.Println("  ✅ SWARM DEMO COMPLETE")
	fmt.Println("═══════════════════════════════════════════════════════════════")
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
