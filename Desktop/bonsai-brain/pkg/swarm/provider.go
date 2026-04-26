package swarm

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/donn/bonsai-brain/pkg/engine"
	"github.com/donn/bonsai-brain/pkg/openai"
)

// ProviderType identifies the backend API provider.
type ProviderType string

const (
	ProviderOpenRouter ProviderType = "openrouter"
	ProviderGroq       ProviderType = "groq"
	ProviderNVIDIA     ProviderType = "nvidia"
	ProviderGemini     ProviderType = "gemini"
	ProviderCohere     ProviderType = "cohere"
	ProviderOpenAI     ProviderType = "openai"
	ProviderOllama     ProviderType = "ollama"
	ProviderLocal      ProviderType = "local"
)

// ProviderConfig holds endpoint and auth for a single provider.
type ProviderConfig struct {
	Type       ProviderType
	BaseURL    string
	APIKey     string
	APIKeyEnv  string
	Models     []string
	RateLimit  int // requests per minute
	TimeoutSec int
}

// FreeTierModels returns the known free-tier models per provider.
func FreeTierModels() map[ProviderType][]string {
	return map[ProviderType][]string{
		ProviderOpenRouter: {
			"qwen/qwen3-coder:free",
			"meta-llama/llama-3.3-70b-instruct:free",
			"meta-llama/llama-3.2-3b-instruct:free",
			"google/gemma-3-27b-it:free",
			"google/gemma-3-12b-it:free",
			"google/gemma-3-4b-it:free",
			"google/gemma-3n-e4b-it:free",
			"google/gemma-3n-e2b-it:free",
			"openai/gpt-oss-120b:free",
			"openai/gpt-oss-20b:free",
			"nvidia/nemotron-3-super-120b-a12b:free",
			"nvidia/nemotron-3-nano-30b-a3b:free",
			"nvidia/nemotron-nano-12b-v2-vl:free",
			"nvidia/nemotron-nano-9b-v2:free",
			"z-ai/glm-4.5-air:free",
			"liquid/lfm-2.5-1.2b-thinking:free",
			"liquid/lfm-2.5-1.2b-instruct:free",
			"nousresearch/hermes-3-llama-3.1-405b:free",
			"cognitivecomputations/dolphin-mistral-24b-venice-edition:free",
			"inclusionai/ling-2.6-1t:free",
			"inclusionai/ling-2.6-flash:free",
			"tencent/hy3-preview:free",
			"minimax/minimax-m2.5:free",
		},
		ProviderGroq: {
			"llama-3.1-8b-instant",
			"llama-3.3-70b-versatile",
			"qwen/qwen3-32b",
			"meta-llama/llama-4-scout-17b-16e-instruct",
			"groq/compound",
			"groq/compound-mini",
		},
		ProviderGemini: {
			"gemini-2.0-flash",
			"gemini-2.0-flash-lite",
			"gemini-2.5-flash",
			"gemma-3-1b-it",
			"gemma-3-4b-it",
			"gemma-3-12b-it",
			"gemma-3-27b-it",
		},
		ProviderNVIDIA: {
			"deepseek-ai/deepseek-v4-flash",
			"deepseek-ai/deepseek-v4-pro",
			"deepseek-ai/deepseek-v3.2",
			"google/gemma-3-4b-it",
			"google/gemma-3-12b-it",
			"google/gemma-3-27b-it",
		},
		ProviderCohere: {
			"command-r-08-2024",
			"command-r-plus-08-2024",
			"command-r7b-12-2024",
			"command-a-03-2025",
		},
		ProviderOllama: {
			"qwen2.5-tiny",
		},
		ProviderLocal: {
			"prism-bonsai-1.7b",
		},
	}
}

// DefaultProviderConfigs returns provider configs with keys from environment.
func DefaultProviderConfigs() []ProviderConfig {
	cfgs := []ProviderConfig{
		{
			Type:       ProviderOpenRouter,
			BaseURL:    "https://openrouter.ai/api/v1",
			APIKeyEnv:  "OPENROUTER_API_KEY",
			Models:     FreeTierModels()[ProviderOpenRouter],
			RateLimit:  20,
			TimeoutSec: 60,
		},
		{
			Type:       ProviderGroq,
			BaseURL:    "https://api.groq.com/openai/v1",
			APIKeyEnv:  "GROQ_API_KEY",
			Models:     FreeTierModels()[ProviderGroq],
			RateLimit:  30,
			TimeoutSec: 30,
		},
		{
			Type:       ProviderGemini,
			BaseURL:    "https://generativelanguage.googleapis.com/v1beta/openai",
			APIKeyEnv:  "GEMINI_API_KEY",
			Models:     FreeTierModels()[ProviderGemini],
			RateLimit:  60,
			TimeoutSec: 30,
		},
		{
			Type:       ProviderNVIDIA,
			BaseURL:    "https://integrate.api.nvidia.com/v1",
			APIKeyEnv:  "NVIDIA_API_KEY",
			Models:     FreeTierModels()[ProviderNVIDIA],
			RateLimit:  60,
			TimeoutSec: 60,
		},
		{
			Type:       ProviderCohere,
			BaseURL:    "https://api.cohere.com/v1",
			APIKeyEnv:  "COHERE_API_KEY",
			Models:     FreeTierModels()[ProviderCohere],
			RateLimit:  20,
			TimeoutSec: 30,
		},
		{
			Type:       ProviderOllama,
			BaseURL:    "http://127.0.0.1:11434/v1",
			APIKeyEnv:  "",
			Models:     FreeTierModels()[ProviderOllama],
			RateLimit:  1000,
			TimeoutSec: 60,
		},
		{
			Type:       ProviderLocal,
			BaseURL:    "http://127.0.0.1:11434/v1",
			APIKeyEnv:  "",
			Models:     FreeTierModels()[ProviderLocal],
			RateLimit:  1000,
			TimeoutSec: 60,
		},
	}

	// Resolve API keys from environment.
	for i := range cfgs {
		if cfgs[i].APIKeyEnv != "" {
			cfgs[i].APIKey = os.Getenv(cfgs[i].APIKeyEnv)
		}
	}
	return cfgs
}

// ActiveProviders filters configs that have a valid API key or are local.
func ActiveProviders(configs []ProviderConfig) []ProviderConfig {
	var active []ProviderConfig
	for _, c := range configs {
		if c.Type == ProviderOllama || c.Type == ProviderLocal {
			active = append(active, c)
			continue
		}
		if c.APIKey != "" {
			active = append(active, c)
		}
	}
	return active
}

// ProviderRegistry holds loaded providers and can build model clients.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[ProviderType]*ProviderConfig
	clients   map[string]engine.ModelClient
}

// NewProviderRegistry creates a registry from active configs.
func NewProviderRegistry(configs []ProviderConfig) *ProviderRegistry {
	r := &ProviderRegistry{
		providers: make(map[ProviderType]*ProviderConfig),
		clients:   make(map[string]engine.ModelClient),
	}
	for _, c := range configs {
		c := c
		r.providers[c.Type] = &c
	}
	return r
}

// ModelClient returns a cached or new client for the given model ID.
func (r *ProviderRegistry) ModelClient(modelID string) (engine.ModelClient, error) {
	r.mu.RLock()
	if c, ok := r.clients[modelID]; ok {
		r.mu.RUnlock()
		return c, nil
	}
	r.mu.RUnlock()

	// Find which provider owns this model.
	var cfg *ProviderConfig
	for _, c := range r.providers {
		for _, m := range c.Models {
			if m == modelID {
				cfg = c
				break
			}
		}
		if cfg != nil {
			break
		}
	}
	if cfg == nil {
		return nil, fmt.Errorf("unknown model: %s", modelID)
	}

	var client engine.ModelClient
	if cfg.Type == ProviderCohere {
		client = NewCohereClient(cfg.BaseURL, cfg.APIKey, modelID)
	} else {
		oc := openai.NewClient(cfg.BaseURL, cfg.APIKey, modelID)
		if cfg.Type == ProviderOpenRouter {
			oc.ExtraHeaders = map[string]string{
				"HTTP-Referer": "https://github.com/donn/bonsai-brain",
				"X-Title":      "Bonsai Brain Swarm",
			}
		}
		client = oc
	}

	r.mu.Lock()
	r.clients[modelID] = client
	r.mu.Unlock()
	return client, nil
}

// AllModels returns every model ID across all registered providers.
func (r *ProviderRegistry) AllModels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []string
	for _, c := range r.providers {
		out = append(out, c.Models...)
	}
	return out
}

// ProviderFor returns the provider type that serves the given model.
func (r *ProviderRegistry) ProviderFor(modelID string) ProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for t, c := range r.providers {
		for _, m := range c.Models {
			if m == modelID {
				return t
			}
		}
	}
	return ""
}

// HealthCheck pings a provider by making a lightweight chat request.
func (r *ProviderRegistry) HealthCheck(ctx context.Context, modelID string) (latency time.Duration, err error) {
	client, err := r.ModelClient(modelID)
	if err != nil {
		return 0, err
	}
	start := time.Now()
	_, err = client.Stream(ctx, []engine.Message{
		{Role: "user", Content: "Hi"},
	}, nil)
	return time.Since(start), err
}
