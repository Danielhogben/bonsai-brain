package swarm

import (
	"testing"
	"time"
)

func TestSwarmConfigDefaults(t *testing.T) {
	cfg := &SwarmConfig{
		GlobalRetry: GlobalRetryConfig{
			Enabled:         true,
			BaseDelay:       time.Second,
			MaxDelay:        30 * time.Second,
			Multiplier:      2.0,
			Jitter:          0.5,
			MaxRetries:      5,
		},
		Agents: []AgentConfig{},
	}

	if !cfg.GlobalRetry.Enabled {
		t.Error("GlobalRetry should be enabled by default")
	}
	if cfg.GlobalRetry.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", cfg.GlobalRetry.MaxRetries)
	}
	if len(cfg.Agents) != 0 {
		t.Errorf("Agents len = %d, want 0", len(cfg.Agents))
	}
}

func TestAgentConfigDefaults(t *testing.T) {
	ac := AgentConfig{
		Name:     "test-agent",
		Role:     "worker",
		MaxIter:  10,
	}
	if ac.MaxIter != 10 {
		t.Errorf("MaxIter = %d, want 10", ac.MaxIter)
	}
}

func TestToolConfig(t *testing.T) {
	tc := ToolConfig{
		WebSearch: WebSearchConfig{
			Enabled:    true,
			Provider:   "tavily",
			MaxResults: 5,
		},
	}
	if !tc.WebSearch.Enabled {
		t.Error("WebSearch should be enabled")
	}
}
