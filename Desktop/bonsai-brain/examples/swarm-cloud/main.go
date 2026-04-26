// swarm-cloud demonstrates a full distributed cloud stack using every
// available API key and free-tier model. It spawns one sub-agent per model,
// sends a task in parallel, and compares results.
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/donn/bonsai-brain/pkg/agent"
	"github.com/donn/bonsai-brain/pkg/engine"
	"github.com/donn/bonsai-brain/pkg/swarm"
)

func main() {
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("  🌳 BONSAI BRAIN — FULL CLOUD SWARM STACK")
	fmt.Println("  Maxing out every free API key we have...")
	fmt.Println("═══════════════════════════════════════════════════════════════")

	ctx := context.Background()

	// ------------------------------------------------------------------
	// 1. Load all provider configs from environment.
	// ------------------------------------------------------------------
	configs := swarm.DefaultProviderConfigs()
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

	// ------------------------------------------------------------------
	// 2. Build provider registry and spawn agents.
	// ------------------------------------------------------------------
	registry := swarm.NewProviderRegistry(active)
	swarmInst := swarm.NewSwarm(registry)

	fmt.Println("\n🤖 Spawning agents...")
	spawned, err := swarmInst.SpawnAll()
	if err != nil {
		fmt.Printf("❌ Spawn error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Spawned %d agents\n", len(spawned))
	for _, sa := range spawned {
		fmt.Printf("      → %-45s (%s)\n", sa.ID, sa.Model)
	}

	// ------------------------------------------------------------------
	// 3. Health check all agents with a lightweight ping.
	// ------------------------------------------------------------------
	fmt.Println("\n🏥 Health check (sending 'Hi' to each agent)...")
	healthCtx, healthCancel := context.WithTimeout(ctx, 30*time.Second)
	defer healthCancel()

	health := swarmInst.HealthCheckAll(healthCtx)
	healthyCount := 0
	for id, lat := range health {
		if lat < 0 {
			fmt.Printf("   🔴 %-45s DEAD\n", id)
		} else {
			fmt.Printf("   🟢 %-45s %v\n", id, lat)
			healthyCount++
		}
	}
	fmt.Printf("\n   Healthy: %d / %d\n", healthyCount, len(health))

	// ------------------------------------------------------------------
	// 4. Send a task to every healthy agent in parallel.
	// ------------------------------------------------------------------
	if healthyCount == 0 {
		fmt.Println("❌ No healthy agents. Check your API keys and network.")
		os.Exit(1)
	}

	// Task: a simple reasoning question.
	task := swarm.Task{
		ID:      "swarm-demo-1",
		Prompt:  "Explain the concept of 'swarm intelligence' in 2 sentences. Be concise.",
		System:  "You are a helpful assistant. Keep responses under 50 words.",
		MaxIter: 1,
	}

	fmt.Println("\n📨 Dispatching task to all agents:")
	fmt.Printf("   \"%s\"\n", task.Prompt)

	dispatchCtx, dispatchCancel := context.WithTimeout(ctx, 120*time.Second)
	defer dispatchCancel()

	start := time.Now()
	results := swarmInst.Distribute(dispatchCtx, task)
	elapsed := time.Since(start)

	// ------------------------------------------------------------------
	// 5. Print results table.
	// ------------------------------------------------------------------
	fmt.Println("\n═══════════════════════════════════════════════════════════════")
	fmt.Println("  RESULTS")
	fmt.Println("═══════════════════════════════════════════════════════════════")

	// Sort by latency (fastest first).
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

	// ------------------------------------------------------------------
	// 6. Aggregate with strategies.
	// ------------------------------------------------------------------
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

	// ------------------------------------------------------------------
	// 7. Summary stats.
	// ------------------------------------------------------------------
	fmt.Println("\n═══════════════════════════════════════════════════════════════")
	fmt.Println("  SUMMARY")
	fmt.Println("═══════════════════════════════════════════════════════════════")

	success := 0
	fail := 0
	var totalLat time.Duration
	for _, r := range results {
		if r.Error == nil {
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

// ------------------------------------------------------------------
// Unused helpers kept for future expansion.
// ------------------------------------------------------------------

var _ = sync.Mutex{} // suppress unused import
var _ = agent.Config{}
var _ = engine.Message{}
