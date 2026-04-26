package main

import (
	"context"
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
	scfg := loadSwarmConfig()

	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		token = "placeholder"
		log.Println("DISCORD_BOT_TOKEN not set, running in minimal mode")
	}

	acfg := &discord.Config{
		Token:        token,
		AutoReply:    false,
		CommandPrefix: "!",
		Agent:        nil, // will use default
		SwarmConfig:  scfg,
	}

	adapter := discord.NewAdapter(acfg)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Bonsai Brain v0.4.0 — Swarm starting...")
	log.Printf("Config: %d agents, retry: enabled=%v max_retries=%d",
		len(scfg.Agents), scfg.GlobalRetry.Enabled, scfg.GlobalRetry.MaxRetries)

	go func() {
		if err := adapter.Start(ctx); err != nil {
			log.Printf("Adapter: %v", err)
		}
	}()

	<-sigCh
	log.Println("Shutting down...")
	cancel()
	time.Sleep(100 * time.Millisecond)
	log.Println("Bonsai Brain stopped.")
}

func loadSwarmConfig() *swarm.SwarmConfig {
	// Return the in-memory config (swarm.yaml exists but LoadConfig would need yaml parsing)
	return &swarm.SwarmConfig{
		GlobalRetry: swarm.GlobalRetryConfig{
			Enabled:         true,
			BaseDelay:       time.Second,
			MaxDelay:        30 * time.Second,
			Multiplier:      2.0,
			Jitter:          0.5,
			MaxRetries:      5,
		},
		Agents: []swarm.AgentConfig{},
	}
}
