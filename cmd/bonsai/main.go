package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/donn/bonsai-brain/pkg/discord"
	"github.com/donn/bonsai-brain/pkg/swarm"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load swarm config
	scfg, err := swarm.LoadConfig("./pkg/swarm/swarm.yaml")
	if err != nil {
		log.Printf("Warning: could not load swarm config: %v", err)
		scfg = &swarm.SwarmConfig{
			GlobalRetry: swarm.GlobalRetryConfig{
				Enabled:    true,
				BaseDelay:  time.Second,
				MaxDelay:   30 * time.Second,
				Multiplier: 2.0,
				Jitter:     0.5,
				MaxRetries: 5,
			},
			Agents: []swarm.AgentConfig{},
		}
	}

	// Get Discord token
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		token = "test_token_placeholder"
		log.Println("DISCORD_BOT_TOKEN not set, using placeholder")
	}

	// Build agent config
	acfg := &discord.Config{
		Token:     token,
		SwarmCfg:  scfg,
		Agent:     &discord.AgentStub{MaxIter: 10},
	}

	// Create adapter
	adapter := discord.NewAdapter(acfg)

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start adapter
	log.Println("Bonsai Brain v0.4.0 — Swarm starting...")
	log.Printf("Config: %d agents, retry: enabled=%v max_retries=%d",
		len(scfg.Agents), scfg.GlobalRetry.Enabled, scfg.GlobalRetry.MaxRetries)

	go func() {
		if err := adapter.Start(ctx); err != nil {
			log.Printf("Adapter stopped: %v", err)
		}
	}()

	<-sigCh
	log.Println("Shutting down...")
	cancel()
	time.Sleep(100 * time.Millisecond)
	log.Println("Bonsai Brain stopped.")
}
