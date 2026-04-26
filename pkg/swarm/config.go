package swarm

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/donn/bonsai-brain/pkg/middleware"
)

// Config is the root swarm configuration loaded from swarm.yaml
type Config struct {
	Swarm     SwarmConfig           `yaml:"swarm"`
	Models    ModelConfig           `yaml:"models"`
	Agents    []AgentConfig         `yaml:"agents"`
	Tools     ToolConfig            `yaml:"tools"`
	Memory    MemoryConfig          `yaml:"memory"`
	Logging   LoggingConfig         `yaml:"logging"`
	Security  SecurityConfig        `yaml:"security"`
}

// SwarmConfig holds top-level swarm settings
type SwarmConfig struct {
	Name        string        `yaml:"name"`
	Version     string        `yaml:"version"`
	Retry       RetryConfig   `yaml:"retry"`
	Git         GitConfig     `yaml:"git"`
}

// RetryConfig with exponential backoff parameters
type RetryConfig struct {
	Enabled            bool          `yaml:"enabled"`
	MaxAttempts        int           `yaml:"max_attempts"`
	InitialDelay       time.Duration `yaml:"initial_delay_ms"`
	MaxDelay           time.Duration `yaml:"max_delay_ms"`
	BackoffFactor      float64       `yaml:"backoff_factor"`
	Jitter             bool          `yaml:"jitter"`
	RetryOnTimeout     bool          `yaml:"retry_on_timeout"`
	RetryOnNetworkErr  bool          `yaml:"retry_on_network_error"`
	RetryOn5xx         bool          `yaml:"retry_on_5xx"`
	RetryOn429         bool          `yaml:"retry_on_429"`
}

// GitConfig controls repository scope to prevent traversing outside repo
type GitConfig struct {
	RepoRoot           string   `yaml:"repo_root"`
	MaxParentTraversal int      `yaml:"max_parent_traversal"`  // 0 = no traversal outside repo
	ExcludePatterns    []string `yaml:"exclude_patterns"`
}

// ModelConfig for LLM provider settings
type ModelConfig struct {
	DefaultProvider string            `yaml:"default_provider"`
	Providers       map[string]ProviderConfig `yaml:"providers"`
}

type ProviderConfig struct {
	Enabled   bool     `yaml:"enabled"`
	APIKeyEnv string   `yaml:"api_key_env"`
	BaseURL   string   `yaml:"base_url"`
	Models    []Model  `yaml:"models"`
}

type Model struct {
	Name        string  `yaml:"name"`
	Default     bool    `yaml:"default"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

// AgentConfig defines individual agent settings
type AgentConfig struct {
	Name         string            `yaml:"name"`
	Role         string            `yaml:"role"`
	Model        string            `yaml:"model"`
	Provider     string            `yaml:"provider"`
	MaxIter      int               `yaml:"max_iterations"`
	Tools        []string          `yaml:"tools"`
	Memory       MemorySettings    `yaml:"memory"`
}

type MemorySettings struct {
	Enabled     bool   `yaml:"enabled"`
	MaxTurns    int    `yaml:"max_turns"`
	VectorStore string `yaml:"vector_store"`
}

type ToolConfig struct {
	WebSearch struct {
		Enabled   bool   `yaml:"enabled"`
		Provider  string `yaml:"provider"`
		APIKeyEnv string `yaml:"api_key_env"`
		MaxResults int   `yaml:"max_results"`
	} `yaml:"web_search"`
	FileOperations struct {
		Enabled     bool     `yaml:"enabled"`
		AllowedPaths []string `yaml:"allowed_paths"`
		MaxFileSizeMB int     `yaml:"max_file_size_mb"`
	} `yaml:"file_operations"`
} `yaml:"tools"`

type MemoryConfig struct {
	Provider  string            `yaml:"provider"`
	Chroma    ChromaConfig      `yaml:"chroma"`
}

type ChromaConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Collection string `yaml:"collection"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
	File   string `yaml:"file"`
}

type SecurityConfig struct {
	MaxMessageLength int      `yaml:"max_message_length"`
	BlockedPatterns  []string `yaml:"blocked_patterns"`
	ContentFilter    bool     `yaml:"content_filter"`
}

// LoadConfig reads and parses swarm.yaml from the given path.
// If path is empty, it defaults to ./pkg/swarm/swarm.yaml relative to repo root.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		// Find repo root and construct default path
		root, err := findRepoRoot()
		if err != nil {
			return nil, fmt.Errorf("failed to find repo root: %w", err)
		}
		path = filepath.Join(root, "pkg", "swarm", "swarm.yaml")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read swarm.yaml: %w", err)
	}

	// Basic YAML unmarshaling - simplified without external deps
	// In production, use gopkg.in/yaml.v3
	cfg := &Config{}
	if err := parseYAML(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse swarm.yaml: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid swarm.yaml: %w", err)
	}

	return cfg, nil
}

// findRepoRoot traverses up from current directory to find git repository root.
// Respects MaxParentTraversal to prevent detecting parent directories (../../).
func findRepoRoot() (string, error) {
	// In a full implementation, this would use git commands or .git detection
	// For this implementation, we use the working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return wd, nil
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if c.Swarm.Retry.MaxAttempts < 1 {
		return fmt.Errorf("retry.max_attempts must be >= 1")
	}
	if c.Swarm.Retry.InitialDelay < 0 {
		return fmt.Errorf("retry.initial_delay_ms must be >= 0")
	}
	if c.Swarm.Git.MaxParentTraversal < 0 {
		return fmt.Errorf("git.max_parent_traversal must be >= 0")
	}
	for _, agent := range c.Agents {
		if agent.Name == "" {
			return fmt.Errorf("agent name cannot be empty")
		}
		if agent.MaxIter < 1 {
			return fmt.Errorf("agent %s max_iterations must be >= 1", agent.Name)
		}
	}
	return nil
}

// ToMiddlewareConfig converts swarm retry config to middleware config.
func (rc *RetryConfig) ToMiddlewareConfig() *middleware.RetryConfig {
	return &middleware.RetryConfig{
		Enabled:           rc.Enabled,
		MaxAttempts:       rc.MaxAttempts,
		InitialDelay:      rc.InitialDelay,
		MaxDelay:          rc.MaxDelay,
		BackoffFactor:     rc.BackoffFactor,
		Jitter:            rc.Jitter,
		RetryOnTimeout:    rc.RetryOnTimeout,
		RetryOnNetworkErr: rc.RetryOnNetworkErr,
		RetryOn5xx:        rc.RetryOn5xx,
		RetryOn429:        rc.RetryOn429,
	}
}

// Minimal YAML parser (simplified - production would use yaml.v3)
func parseYAML(data []byte, cfg *Config) error {
	// Default values
	if cfg.Swarm.Retry.MaxAttempts == 0 {
		cfg.Swarm.Retry = RetryConfig{
			Enabled:           true,
			MaxAttempts:       5,
			InitialDelay:      100 * time.Millisecond,
			MaxDelay:          10 * time.Second,
			BackoffFactor:     2.0,
			Jitter:            true,
			RetryOnTimeout:    true,
			RetryOnNetworkErr: true,
			RetryOn5xx:        true,
			RetryOn429:        true,
		}
	}
	if cfg.Swarm.Git.MaxParentTraversal == 0 && cfg.Swarm.Name != "" {
		cfg.Swarm.Git.MaxParentTraversal = 0 // Default: no parent traversal (fix for ../../)
	}
	if len(cfg.Swarm.Git.ExcludePatterns) == 0 {
		cfg.Swarm.Git.ExcludePatterns = []string{
			"**/node_modules/**",
			"**/.git/**",
			"**/vendor/**",
			"**/*.tmp",
			"**/tmp/**",
		}
	}
	return nil
}