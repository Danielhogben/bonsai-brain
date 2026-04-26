package swarm

import "time"

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

type WebSearchConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Provider   string `yaml:"provider"`
	APIKeyEnv  string `yaml:"api_key_env"`
	MaxResults int    `yaml:"max_results"`
}

type FileOpsConfig struct {
	Enabled        bool     `yaml:"enabled"`
	AllowedPaths   []string `yaml:"allowed_paths"`
	MaxFileSizeMB  int      `yaml:"max_file_size_mb"`
}

type ToolConfig struct {
	WebSearch    WebSearchConfig    `yaml:"web_search"`
	FileOperations FileOpsConfig     `yaml:"file_operations"`
}

type MemoryConfig struct {
	Provider  string        `yaml:"provider"`
	Chroma    ChromaConfig  `yaml:"chroma"`
}

type ChromaConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Collection string `yaml:"collection"`
}

type GlobalRetryConfig struct {
	Enabled         bool          `yaml:"enabled"`
	BaseDelay       time.Duration `yaml:"base_delay"`
	MaxDelay        time.Duration `yaml:"max_delay"`
	Multiplier      float64       `yaml:"multiplier"`
	Jitter          float64       `yaml:"jitter"`
	MaxRetries      int           `yaml:"max_retries"`
}

type SwarmConfig struct {
	GlobalRetry GlobalRetryConfig    `yaml:"global_retry"`
	Agents      []AgentConfig        `yaml:"agents"`
	ToolConfig  ToolConfig          `yaml:"tool_config"`
	Memory      MemoryConfig        `yaml:"memory"`
}
