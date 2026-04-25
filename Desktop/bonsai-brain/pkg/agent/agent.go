// Package agent provides a hierarchical agent inspired by agent-zero and
// deer-flow patterns.  An Agent wraps a QueryEngine with middleware and
// guardrail pipelines, a context, a configuration, and optional parent
// reference for tree-structured delegation (sub-agents).
package agent

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/donn/bonsai-brain/pkg/engine"
	"github.com/donn/bonsai-brain/pkg/guardrail"
	"github.com/donn/bonsai-brain/pkg/middleware"
)

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

// Config holds agent-level configuration knobs.
type Config struct {
	Name         string
	SystemPrompt string
	MaxDepth     int // max sub-agent depth; 0 = unlimited
	MaxIter      int // max engine iterations per turn; 0 = default (10)
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(name string) Config {
	return Config{
		Name:    name,
		MaxIter: 10,
	}
}

// ---------------------------------------------------------------------------
// RuntimeFeatures
// ---------------------------------------------------------------------------

// RuntimeFeatures is a set of boolean flags that control which capabilities
// are active for an agent at runtime.
type RuntimeFeatures struct {
	Sandbox       bool // restrict file/network access
	Memory        bool // enable long-term memory
	Summarization bool // auto-summarize long conversations
	SubAgent      bool // allow spawning child agents
	Vision        bool // enable image/vision inputs
}

// ---------------------------------------------------------------------------
// Agent
// ---------------------------------------------------------------------------

// Agent is a hierarchical reasoning unit.  It holds a depth counter (for
// guarding against infinite recursion), a configuration, a context bag,
// and an optional parent reference.
type Agent struct {
	Depth  int
	Config Config
	Ctx    *AgentContext
	Parent *Agent

	Engine   *engine.QueryEngine
	Features RuntimeFeatures

	InMiddleware  *middleware.InputPipeline
	OutMiddleware *middleware.OutputPipeline
	InGuardrails  *guardrail.InputPipeline
	OutGuardrails *guardrail.OutputPipeline

	// counter for unique sub-agent IDs
	idSeq atomic.Int64
}

// AgentContext is the per-agent mutable state carried through a turn.
type AgentContext struct {
	ID      string
	History []engine.Message
	State   map[string]any
}

// New creates a root agent (depth 0).
func New(cfg Config, eng *engine.QueryEngine) *Agent {
	return &Agent{
		Depth:  0,
		Config: cfg,
		Engine: eng,
		Ctx: &AgentContext{
			ID:    cfg.Name + "-0",
			State: make(map[string]any),
		},
		InMiddleware:  middleware.NewInputPipeline(),
		OutMiddleware: middleware.NewOutputPipeline(),
		InGuardrails:  guardrail.NewInputPipeline(),
		OutGuardrails: guardrail.NewOutputPipeline(),
	}
}

// ---------------------------------------------------------------------------
// Sub-agent spawning
// ---------------------------------------------------------------------------

// SpawnSubAgent creates a child agent with depth+1.  It returns an error if
// the configured MaxDepth would be exceeded.
func (a *Agent) SpawnSubAgent(cfg Config, eng *engine.QueryEngine) (*Agent, error) {
	if a.Config.MaxDepth > 0 && a.Depth+1 > a.Config.MaxDepth {
		return nil, fmt.Errorf("agent: max depth %d exceeded", a.Config.MaxDepth)
	}
	child := &Agent{
		Depth:  a.Depth + 1,
		Config: cfg,
		Parent: a,
		Engine: eng,
		Ctx: &AgentContext{
			ID:    fmt.Sprintf("%s-%d", cfg.Name, a.idSeq.Add(1)),
			State: make(map[string]any),
		},
		InMiddleware:  middleware.NewInputPipeline(),
		OutMiddleware: middleware.NewOutputPipeline(),
		InGuardrails:  guardrail.NewInputPipeline(),
		OutGuardrails: guardrail.NewOutputPipeline(),
		Features:      a.Features, // inherit features
	}
	return child, nil
}

// ---------------------------------------------------------------------------
// GenerateText
// ---------------------------------------------------------------------------

// GenerateText runs the full pipeline:
//
//	input middleware → input guardrails → engine → output guardrails → output middleware
//
// It returns the final text or an error.
func (a *Agent) GenerateText(ctx context.Context, userMessage string) (string, error) {
	// 1. Input middleware
	processed, err := a.InMiddleware.Run(ctx, userMessage)
	if err != nil {
		return "", fmt.Errorf("agent %q input middleware: %w", a.Config.Name, err)
	}

	// 2. Input guardrails
	igResult := a.InGuardrails.Run(ctx, processed)
	if !igResult.Pass {
		return "", fmt.Errorf("agent %q input guardrail: %s (%s)", a.Config.Name, igResult.Action, igResult.Message)
	}

	// 3. Build messages and run engine
	messages := a.buildMessages(ctx, processed)
	maxIter := a.Config.MaxIter
	if maxIter <= 0 {
		maxIter = 10
	}
	resp, err := a.Engine.Run(ctx, messages, maxIter)
	if err != nil {
		return "", fmt.Errorf("agent %q engine: %w", a.Config.Name, err)
	}

	// 4. Output guardrails
	ogOutput, ogResult := a.OutGuardrails.Run(ctx, resp.Content)
	if !ogResult.Pass {
		return "", fmt.Errorf("agent %q output guardrail: %s (%s)", a.Config.Name, ogResult.Action, ogResult.Message)
	}

	// 5. Output middleware
	final, err := a.OutMiddleware.Run(ctx, ogOutput)
	if err != nil {
		// If it's an abort with retry, surface a clear message.
		if abortErr, ok := err.(*middleware.AbortError); ok {
			if abortErr.Retry {
				return "", fmt.Errorf("agent %q output middleware requested retry: %s", a.Config.Name, abortErr.Reason)
			}
			return final, nil // non-retryable abort returns partial output
		}
		return "", fmt.Errorf("agent %q output middleware: %w", a.Config.Name, err)
	}

	// 6. Update history
	a.Ctx.History = append(a.Ctx.History,
		engine.Message{Role: "user", Content: userMessage},
		engine.Message{Role: "assistant", Content: final},
	)

	return final, nil
}

// GenerateWithRetry runs GenerateText with automatic retries on retryable
// output middleware aborts.
func (a *Agent) GenerateWithRetry(ctx context.Context, userMessage string, maxRetries int) (string, error) {
	return middleware.RunWithRetry(ctx, maxRetries, 0, func(ctx context.Context) (string, error) {
		return a.GenerateText(ctx, userMessage)
	})
}

// buildMessages constructs the message slice for the engine, including the
// system prompt and conversation history.
func (a *Agent) buildMessages(ctx context.Context, userMessage string) []engine.Message {
	var messages []engine.Message

	// System prompt
	sysPrompt := a.Config.SystemPrompt
	if sysPrompt != "" {
		messages = append(messages, engine.Message{Role: "system", Content: sysPrompt})
	}

	// History
	messages = append(messages, a.Ctx.History...)

	// Current user message
	messages = append(messages, engine.Message{Role: "user", Content: userMessage})

	return messages
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// Root returns the root agent by walking up the Parent chain.
func (a *Agent) Root() *Agent {
	for a.Parent != nil {
		a = a.Parent
	}
	return a
}

// Ancestors returns the chain of agents from this agent up to (but not
// including) the root.
func (a *Agent) Ancestors() []*Agent {
	var ancestors []*Agent
	cur := a.Parent
	for cur != nil {
		ancestors = append(ancestors, cur)
		cur = cur.Parent
	}
	return ancestors
}

// String returns a human-readable identifier for the agent.
func (a *Agent) String() string {
	return fmt.Sprintf("Agent(%s, depth=%d)", a.Config.Name, a.Depth)
}
