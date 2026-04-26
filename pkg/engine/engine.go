package engine

import "context"

// Message represents a message in the agent interaction
type Message struct {
	Role    string
	Content string
}

// Response represents the agent response
type Response struct {
	Content  string
	Model    string
	ToolUsed string
}

// Runner is the interface for running the agent
type Runner interface {
	Run(ctx context.Context, messages []Message, maxIter int) (*Response, error)
}

type Engine struct{}

// Run executes agent interaction
func (e *Engine) Run(ctx context.Context, messages []Message, maxIter int) (*Response, error) {
	return &Response{
		Content: "Agent response placeholder - engine not implemented",
		Model:   "placeholder",
	}, nil
}
