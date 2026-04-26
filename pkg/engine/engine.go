package engine

import (
	"context"
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Permission pipeline
// ---------------------------------------------------------------------------

// PermissionDecision is a 3-state gate for tool execution.
type PermissionDecision int

const (
	PermissionAllow    PermissionDecision = iota // proceed without asking
	PermissionBlock                              // hard deny
	PermissionAskUser                            // surface to user for approval
)

// PermissionChecker inspects a tool call and returns a decision.
// Returning PermissionAskUser causes the engine to invoke the AskUserFunc.
type PermissionChecker func(call ToolCall) PermissionDecision

// AskUserFunc is called when a tool call is gated behind PermissionAskUser.
// It should return true to proceed, false to block.
type AskUserFunc func(call ToolCall) bool

// ---------------------------------------------------------------------------
// System-prompt builder
// ---------------------------------------------------------------------------

// PromptSource is a labelled chunk of system-prompt text.
type PromptSource struct {
	Label   string // human-readable label used for injection filtering
	Content string
}

// SystemPromptBuilder assembles a system prompt from up to three sources
// and supports an injection filter that can selectively include/exclude
// sources by label.
type SystemPromptBuilder struct {
	sources []PromptSource
}

// NewSystemPromptBuilder creates an empty builder.
func NewSystemPromptBuilder() *SystemPromptBuilder {
	return &SystemPromptBuilder{}
}

// Add appends a named prompt source.
func (b *SystemPromptBuilder) Add(label, content string) *SystemPromptBuilder {
	b.sources = append(b.sources, PromptSource{Label: label, Content: content})
	return b
}

// Build concatenates all sources whose labels are in include (or all sources
// if include is empty).  The resulting string is returned.
func (b *SystemPromptBuilder) Build(include ...string) string {
	allowed := map[string]bool{}
	for _, l := range include {
		allowed[l] = true
	}
	var parts []string
	for _, s := range b.sources {
		if len(allowed) > 0 && !allowed[s.Label] {
			continue
		}
		parts = append(parts, strings.TrimSpace(s.Content))
	}
	return strings.Join(parts, "\n\n")
}

// Default labels for the three canonical sources.
const (
	LabelDefaults    = "defaults"
	LabelUserContext = "user_context"
	LabelSysContext  = "sys_context"
)

// NewDefaultPromptBuilder creates a builder pre-loaded with the three
// canonical sources.  Any source may be empty.
func NewDefaultPromptBuilder(defaults, userCtx, sysCtx string) *SystemPromptBuilder {
	b := NewSystemPromptBuilder()
	if defaults != "" {
		b.Add(LabelDefaults, defaults)
	}
	if userCtx != "" {
		b.Add(LabelUserContext, userCtx)
	}
	if sysCtx != "" {
		b.Add(LabelSysContext, sysCtx)
	}
	return b
}

// ---------------------------------------------------------------------------
// QueryEngine
// ---------------------------------------------------------------------------

// ToolEntry pairs a schema with its executor.
type ToolEntry struct {
	Schema   ToolSchema
	Executor ToolExecutor
}

// QueryEngine is the core reasoning loop.  It holds a model client, a tool
// registry, a permission policy, and configuration knobs.
type QueryEngine struct {
	Model      ModelClient
	Tools      map[string]ToolEntry
	Permission PermissionChecker
	AskUser    AskUserFunc
	MaxTokens  int // reserved for future use; not yet forwarded to the model

	// PromptBuilder, if non-nil, is used to build the system prompt for
	// each Run invocation.  The caller can also pass a pre-built system
	// message in the messages slice; the builder result is prepended.
	PromptBuilder *SystemPromptBuilder
	// PromptInclude controls which PromptSource labels are included when
	// building the system prompt.  If empty, all sources are included.
	PromptInclude []string
}

// NewQueryEngine creates an engine with sensible defaults.
func NewQueryEngine(model ModelClient) *QueryEngine {
	return &QueryEngine{
		Model: model,
		Tools: make(map[string]ToolEntry),
		Permission: func(_ ToolCall) PermissionDecision {
			return PermissionAllow // default: allow everything
		},
	}
}

// RegisterTool adds a tool to the engine's registry.
func (e *QueryEngine) RegisterTool(schema ToolSchema, exec ToolExecutor) {
	e.Tools[schema.Name] = ToolEntry{Schema: schema, Executor: exec}
}

// ToolSchemas returns the list of schemas to advertise to the model.
func (e *QueryEngine) ToolSchemas() []ToolSchema {
	out := make([]ToolSchema, 0, len(e.Tools))
	for _, t := range e.Tools {
		out = append(out, t.Schema)
	}
	return out
}

// Run executes the reasoning loop:
//
//	1. Prepend a system prompt (if builder is configured).
//	2. Call the model.
//	3. If the model returns a stop finish reason, return the response.
//	4. For each tool call, check permissions → execute or block → append
//	   the result as a tool message and loop.
//
// maxIterations limits the number of model calls to prevent runaway loops.
// A value <= 0 defaults to 10.
func (e *QueryEngine) Run(ctx context.Context, messages []Message, maxIterations int) (*Response, error) {
	if maxIterations <= 0 {
		maxIterations = 10
	}

	// Build system prompt from builder if configured.
	if e.PromptBuilder != nil {
		sysText := e.PromptBuilder.Build(e.PromptInclude...)
		if sysText != "" {
			sysMsg := Message{Role: "system", Content: sysText}
			// Prepend: system message should be first.
			messages = append([]Message{sysMsg}, messages...)
		}
	}

	tools := e.ToolSchemas()

	for i := 0; i < maxIterations; i++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("engine: context cancelled on iteration %d: %w", i, err)
		}

		resp, err := e.Model.Stream(ctx, messages, tools)
		if err != nil {
			return nil, fmt.Errorf("engine: model call failed on iteration %d: %w", i, err)
		}

		// Append the assistant message so the conversation history grows.
		assistantMsg := Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		messages = append(messages, assistantMsg)

		// If the model is done, return.
		if resp.FinishReason == "stop" || len(resp.ToolCalls) == 0 {
			return resp, nil
		}

		// Process tool calls.
		for _, call := range resp.ToolCalls {
			result, toolErr := e.handleToolCall(ctx, call)
			toolMsg := Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: call.ID,
			}
			if toolErr != nil {
				toolMsg.Content = fmt.Sprintf("error: %s", toolErr)
			}
			messages = append(messages, toolMsg)
		}
	}

	// Exhausted iterations — return the last response as-is.
	return &Response{
		Content:      "max iterations reached",
		FinishReason: "max_iterations",
	}, nil
}

// handleToolCall checks permissions and either executes or blocks a tool call.
func (e *QueryEngine) handleToolCall(ctx context.Context, call ToolCall) (string, error) {
	decision := PermissionAllow
	if e.Permission != nil {
		decision = e.Permission(call)
	}

	switch decision {
	case PermissionAllow:
		// proceed
	case PermissionBlock:
		return "", fmt.Errorf("tool %q blocked by permission policy", call.Name)
	case PermissionAskUser:
		if e.AskUser == nil {
			return "", fmt.Errorf("tool %q requires user approval but no AskUser func configured", call.Name)
		}
		if !e.AskUser(call) {
			return "", fmt.Errorf("tool %q blocked by user", call.Name)
		}
	default:
		return "", fmt.Errorf("unknown permission decision %d for tool %q", decision, call.Name)
	}

	entry, ok := e.Tools[call.Name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %q", call.Name)
	}

	return entry.Executor(ctx, call.Args)
}
