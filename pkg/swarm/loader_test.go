package swarm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSwarmYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "swarm.yaml")
	content := `
swarm:
  name: "test-swarm"
  version: "0.4.0"
  retry:
    enabled: true
    max_attempts: 3
    initial_delay_ms: 50
    max_delay_ms: 5000
    backoff_factor: 1.5
    jitter: false
    retry_on_timeout: true
    retry_on_network_error: true
    retry_on_5xx: true
    retry_on_429: true
  git:
    repo_root: "."
    max_parent_traversal: 0
    exclude_patterns:
      - "**/node_modules/**"
  models:
    default_provider: "openrouter"
    providers:
      openrouter:
        enabled: true
        api_key_env: "OPENROUTER_API_KEY"
        base_url: "https://openrouter.ai/api/v1"
        models:
          - name: "meta-llama/llama-3.3-70b-instruct:free"
            default: true
            max_tokens: 8192
            temperature: 0.7
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadSwarmYAML(path)
	if err != nil {
		t.Fatalf("LoadSwarmYAML failed: %v", err)
	}

	if cfg.Swarm.Name != "test-swarm" {
		t.Errorf("Name = %q, want test-swarm", cfg.Swarm.Name)
	}
	if cfg.Swarm.Version != "0.4.0" {
		t.Errorf("Version = %q, want 0.4.0", cfg.Swarm.Version)
	}
	if !cfg.Swarm.Retry.Enabled {
		t.Error("Retry.Enabled should be true")
	}
	if cfg.Swarm.Retry.MaxAttempts != 3 {
		t.Errorf("Retry.MaxAttempts = %d, want 3", cfg.Swarm.Retry.MaxAttempts)
	}
	if len(cfg.Swarm.Git.ExcludePatterns) != 1 {
		t.Errorf("Git.ExcludePatterns len = %d, want 1", len(cfg.Swarm.Git.ExcludePatterns))
	}
	if cfg.Swarm.Models.DefaultProvider != "openrouter" {
		t.Errorf("DefaultProvider = %q, want openrouter", cfg.Swarm.Models.DefaultProvider)
	}

	openrouter, ok := cfg.Swarm.Models.Providers["openrouter"]
	if !ok {
		t.Fatal("missing openrouter provider")
	}
	if !openrouter.Enabled {
		t.Error("openrouter should be enabled")
	}
	if len(openrouter.Models) != 1 {
		t.Errorf("openrouter models = %d, want 1", len(openrouter.Models))
	}
	if openrouter.Models[0].Name != "meta-llama/llama-3.3-70b-instruct:free" {
		t.Errorf("model name = %q", openrouter.Models[0].Name)
	}
}

func TestToProviderConfigs(t *testing.T) {
	cfg := &SwarmYAML{
		Swarm: SwarmSpec{
			Models: ModelsSpec{
				Providers: map[string]ProviderSpec{
					"openrouter": {
						Enabled:   true,
						APIKeyEnv: "OPENROUTER_API_KEY",
						BaseURL:   "https://openrouter.ai/api/v1",
						Models:    []ModelSpec{{Name: "model-a"}},
					},
					"groq": {
						Enabled:   true,
						APIKeyEnv: "GROQ_API_KEY",
						Models:    []ModelSpec{{Name: "model-b"}},
					},
					"disabled": {
						Enabled: false,
						Models:  []ModelSpec{{Name: "model-c"}},
					},
				},
			},
		},
	}
	configs := cfg.ToProviderConfigs()
	if len(configs) != 2 {
		t.Fatalf("configs = %d, want 2", len(configs))
	}
	types := make(map[ProviderType]bool)
	for _, c := range configs {
		types[c.Type] = true
	}
	if !types[ProviderOpenRouter] {
		t.Error("missing OpenRouter provider")
	}
	if !types[ProviderGroq] {
		t.Error("missing Groq provider")
	}
}
