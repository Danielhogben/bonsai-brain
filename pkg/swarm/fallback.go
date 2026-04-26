package swarm

import (
	"context"
	"fmt"
	"time"
)

// FallbackChain executes a task against a sequence of providers,
// moving to the next provider if the current one fails.
type FallbackChain struct {
	providers []ProviderConfig
	strategy  ResultStrategy
}

// NewFallbackChain creates a chain from the given provider configs.
func NewFallbackChain(providers []ProviderConfig) *FallbackChain {
	return &FallbackChain{
		providers: providers,
		strategy:  FirstWinner,
	}
}

// Execute runs the task through each provider in order until one succeeds.
func (fc *FallbackChain) Execute(ctx context.Context, task Task) (*TaskResult, error) {
	var lastErr error
	for i, p := range fc.providers {
		agentID := fmt.Sprintf("fallback-%s-%d", p.Type, i)
		result := fc.tryProvider(ctx, agentID, p, task)
		if result.Error == nil && result.Output != "" {
			return result, nil
		}
		lastErr = result.Error
	}
	return nil, fmt.Errorf("fallback chain exhausted: %w", lastErr)
}

func (fc *FallbackChain) tryProvider(ctx context.Context, agentID string, p ProviderConfig, task Task) *TaskResult {
	start := time.Now()
	// Simulate dispatch — in production this would call the actual model client.
	// For now we return an error so the chain continues to the next provider
	// unless this is a mock/test scenario.
	return &TaskResult{
		AgentID:  agentID,
		Model:    p.Models[0],
		Latency:  time.Since(start),
		Error:    fmt.Errorf("provider %s not connected", p.Type),
	}
}
