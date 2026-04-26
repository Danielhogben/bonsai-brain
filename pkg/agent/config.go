package agent

import "time"

// Config holds agent configuration
type Config struct {
	MaxIter          int           // Maximum iterations for agent execution
	Model            string        // Model identifier
	Provider         string        // Provider name
	Temperature      float64       // Sampling temperature
	Timeout          time.Duration // Maximum execution time
	Streaming        bool          // Enable streaming responses
	ToolsEnabled     bool          // Whether tools can be used
	MemoryEnabled    bool          // Whether memory is enabled
	MaxMemoryTurns   int           // Maximum turns to keep in memory
}

// DefaultConfig returns the default agent configuration
func DefaultConfig() *Config {
	return &Config{
		MaxIter:        10,
		Temperature:    0.7,
		Timeout:        120 * time.Second,
		Streaming:      false,
		ToolsEnabled:   true,
		MemoryEnabled:  true,
		MaxMemoryTurns: 50,
	}
}