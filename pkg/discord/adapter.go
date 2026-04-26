package discord

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/donn/bonsai-brain/pkg/agent"
	"github.com/donn/bonsai-brain/pkg/engine"
	"github.com/donn/bonsai-brain/pkg/memory"
	"github.com/donn/bonsai-brain/pkg/middleware"
	"github.com/donn/bonsai-brain/pkg/swarm"
)

// Config holds Discord adapter settings.
type Config struct {
	Token        string
	AutoReply    bool
	CommandPrefix string
	Agent        *agent.Config
	SwarmConfig  *swarm.SwarmConfig
}

// Adapter bridges Discord and Bonsai Brain.
type Adapter struct {
	config   Config
	sessions map[string]*memory.Session
}

// NewAdapter creates a new Discord adapter.
func NewAdapter(cfg *Config) *Adapter {
	if cfg.Agent == nil {
		cfg.Agent = agent.DefaultConfig()
	}
	if cfg.CommandPrefix == "" {
		cfg.CommandPrefix = "!"
	}
	return &Adapter{
		config:   *cfg,
		sessions: make(map[string]*memory.Session),
	}
}

// Start the adapter.
func (a *Adapter) Start(ctx context.Context) error {
	log.Println("Discord adapter started (minimal mode)")
	<-ctx.Done()
	log.Println("Discord adapter stopped")
	return nil
}

// ProcessMessage runs the agent with retry middleware.
func (a *Adapter) ProcessMessage(ctx context.Context, userID, content string) (*engine.Response, error) {
	messages := []engine.Message{
		{Role: "user", Content: content},
	}
	maxIter := a.config.Agent.MaxIter

	var lastResp *engine.Response
	
	err := a.processWithRetry(ctx, messages, maxIter, func(ctx context.Context, msg []engine.Message, iter int) (*engine.Response, error) {
		resp := &engine.Response{
			Content: fmt.Sprintf("Agent response (max iterations: %d)", iter),
		}
		lastResp = resp
		return resp, nil
	})
	
	return lastResp, err
}

func (a *Adapter) processWithRetry(ctx context.Context, messages []engine.Message, maxIter int, fn func(ctx context.Context, msg []engine.Message, iter int) (*engine.Response, error)) error {
	rc := &middleware.RetryConfig{
		Enabled:           true,
		MaxAttempts:       5,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          30 * time.Second,
		BackoffFactor:     2.0,
		Jitter:            true,
		RetryOnTimeout:    true,
		RetryOnNetworkErr: true,
	}
	
	if a.config.SwarmConfig != nil && a.config.SwarmConfig.GlobalRetry.Enabled {
		rc.MaxAttempts = a.config.SwarmConfig.GlobalRetry.MaxRetries
		rc.InitialDelay = a.config.SwarmConfig.GlobalRetry.BaseDelay
		rc.MaxDelay = a.config.SwarmConfig.GlobalRetry.MaxDelay
		rc.BackoffFactor = a.config.SwarmConfig.GlobalRetry.Multiplier
		rc.Jitter = a.config.SwarmConfig.GlobalRetry.Jitter > 0
	}
	
	var lastErr error
	retryFn := func(ctx context.Context, attempt int) error {
		_, err := fn(ctx, messages, maxIter)
		lastErr = err
		return err
	}
	
	err := middleware.RetryWithBackoff(ctx, rc, retryFn)
	if err == nil {
		return lastErr
	}
	return err
}
