package engine

import "context"

// Message represents a single message in a conversation with an LLM.
type Message struct {
	Role       string     // "system", "user", "assistant", "tool"
	Content    string
	ToolCalls  []ToolCall // populated when Role == "assistant" and model requests tools
	ToolCallID string     // populated when Role == "tool" to reference the call
}

// ToolCall represents a single tool invocation requested by the model.
type ToolCall struct {
	ID   string         // unique call identifier, echoed back in tool result
	Name string         // tool name to look up in the engine's registry
	Args map[string]any // deserialized arguments
}

// Response is the final output of a single model turn.
type Response struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason string // "stop", "tool_calls", "length", etc.
}

// ToolSchema describes a tool that is advertised to the model.
type ToolSchema struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON-Schema-style parameter description
}

// ModelClient is the interface the engine uses to talk to an LLM provider.
// Implementations must return a fully-populated Response (no streaming
// callback needed at this layer; the engine treats it as a single call).
type ModelClient interface {
	Stream(ctx context.Context, messages []Message, tools []ToolSchema) (*Response, error)
}

// ToolExecutor is the callable signature for a registered tool.
// It receives the deserialized arguments and returns a string result or error.
type ToolExecutor func(ctx context.Context, args map[string]any) (string, error)
