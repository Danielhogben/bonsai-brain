package swarm

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SwarmYAML is the root structure of swarm.yaml
type SwarmYAML struct {
	Swarm SwarmSpec `yaml:"swarm"`
}

type SwarmSpec struct {
	Name    string         `yaml:"name"`
	Version string         `yaml:"version"`
	Retry   RetrySpec      `yaml:"retry"`
	Git     GitSpec        `yaml:"git"`
	Models  ModelsSpec     `yaml:"models"`
}

type RetrySpec struct {
	Enabled               bool    `yaml:"enabled"`
	MaxAttempts           int     `yaml:"max_attempts"`
	InitialDelayMs        int     `yaml:"initial_delay_ms"`
	MaxDelayMs            int     `yaml:"max_delay_ms"`
	BackoffFactor         float64 `yaml:"backoff_factor"`
	Jitter                bool    `yaml:"jitter"`
	RetryOnTimeout        bool    `yaml:"retry_on_timeout"`
	RetryOnNetworkError   bool    `yaml:"retry_on_network_error"`
	RetryOn5xx            bool    `yaml:"retry_on_5xx"`
	RetryOn429            bool    `yaml:"retry_on_429"`
}

type GitSpec struct {
	RepoRoot           string   `yaml:"repo_root"`
	MaxParentTraversal int      `yaml:"max_parent_traversal"`
	ExcludePatterns    []string `yaml:"exclude_patterns"`
}

type ModelsSpec struct {
	DefaultProvider string                    `yaml:"default_provider"`
	Providers       map[string]ProviderSpec   `yaml:"providers"`
}

type ProviderSpec struct {
	Enabled    bool         `yaml:"enabled"`
	APIKeyEnv  string       `yaml:"api_key_env"`
	BaseURL    string       `yaml:"base_url"`
	Models     []ModelSpec  `yaml:"models"`
}

type ModelSpec struct {
	Name        string  `yaml:"name"`
	Default     bool    `yaml:"default"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

// LoadSwarmYAML reads and parses swarm.yaml from the given path.
func LoadSwarmYAML(path string) (*SwarmYAML, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read swarm.yaml: %w", err)
	}
	var cfg SwarmYAML
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse swarm.yaml: %w", err)
	}
	return &cfg, nil
}

// ToProviderConfigs converts SwarmYAML provider definitions into runtime ProviderConfig slices.
func (s *SwarmYAML) ToProviderConfigs() []ProviderConfig {
	var configs []ProviderConfig
	for name, p := range s.Swarm.Models.Providers {
		if !p.Enabled {
			continue
		}
		var models []string
		for _, m := range p.Models {
			models = append(models, m.Name)
		}
		apiKey := ""
		if p.APIKeyEnv != "" {
			apiKey = os.Getenv(p.APIKeyEnv)
		}
		pt := ProviderOpenRouter
		switch name {
		case "groq":
			pt = ProviderGroq
		case "ollama":
			pt = ProviderOllama
		case "local":
			pt = ProviderLocal
		}
		configs = append(configs, ProviderConfig{
			Type:      pt,
			APIKey:    apiKey,
			BaseURL:   p.BaseURL,
			Models:    models,
			RateLimit: 10,
			TimeoutSec: 60,
		})
	}
	return configs
}
