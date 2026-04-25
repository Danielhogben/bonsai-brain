// Package tool provides typed tool definitions with validation, approval gates,
// and lifecycle hooks for the Bonsai Brain agent system.
package tool

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// ToolParam describes a single parameter accepted by a tool.
type ToolParam struct {
	Name        string
	Type        string // "string", "number", "boolean", "object", "array"
	Description string
	Required    bool
}

// ToolHook is a callback invoked before (OnStart) or after (OnEnd) a tool executes.
// Returning a non-nil error from OnStart prevents execution; from OnEnd it is logged but ignored.
type ToolHook func(ctx context.Context, tool string, args map[string]any) error

// ExecuteFunc is the function signature for tool execution.
type ExecuteFunc func(ctx context.Context, args map[string]any) (any, error)

// NeedsApprovalFunc returns true when the given arguments require human approval.
type NeedsApprovalFunc func(args map[string]any) bool

// Tool defines a callable tool with typed parameters, optional approval gate,
// and before/after lifecycle hooks.
type Tool struct {
	Name        string
	Description string
	Parameters  []ToolParam
	Execute     ExecuteFunc

	// NeedsApproval, when non-nil, is called to decide if execution needs human approval.
	NeedsApproval NeedsApprovalFunc

	// OnStart is called before Execute. If it returns an error, execution is aborted.
	OnStart ToolHook

	// OnEnd is called after Execute (even if Execute returned an error).
	OnEnd ToolHook
}

// Validate checks the supplied arguments against the tool's parameter definitions.
// It returns an error describing the first problem found, or nil if everything is valid.
func (t *Tool) Validate(args map[string]any) error {
	paramMap := make(map[string]ToolParam, len(t.Parameters))
	for _, p := range t.Parameters {
		paramMap[p.Name] = p
	}

	// Check required parameters are present.
	for _, p := range t.Parameters {
		if p.Required {
			val, ok := args[p.Name]
			if !ok || val == nil {
				return fmt.Errorf("missing required parameter %q", p.Name)
			}
		}
	}

	// Check types of supplied parameters.
	for name, val := range args {
		p, known := paramMap[name]
		if !known {
			continue // extra keys are allowed, tools can ignore them
		}
		if err := checkType(p.Type, val); err != nil {
			return fmt.Errorf("parameter %q: %w", name, err)
		}
	}

	return nil
}

// Run validates arguments, runs hooks, checks approval, and finally executes the tool.
// It is safe for concurrent use.
func (t *Tool) Run(ctx context.Context, args map[string]any) (any, error) {
	if err := t.Validate(args); err != nil {
		return nil, fmt.Errorf("tool %q validation: %w", t.Name, err)
	}

	// Approval gate
	if t.NeedsApproval != nil && t.NeedsApproval(args) {
		return nil, &ApprovalError{Tool: t.Name, Args: args}
	}

	// OnStart hook
	if t.OnStart != nil {
		if err := t.OnStart(ctx, t.Name, args); err != nil {
			return nil, fmt.Errorf("tool %q hook on_start: %w", t.Name, err)
		}
	}

	// Execute
	result, execErr := t.Execute(ctx, args)

	// OnEnd hook (always runs)
	if t.OnEnd != nil {
		if err := t.OnEnd(ctx, t.Name, args); err != nil {
			// Log-style: we don't fail the run because of an OnEnd error,
			// but we surface it wrapped alongside any execution error.
			if execErr != nil {
				return result, fmt.Errorf("tool %q: %w (on_end: %v)", t.Name, execErr, err)
			}
			return result, fmt.Errorf("tool %q on_end hook: %w", t.Name, err)
		}
	}

	if execErr != nil {
		return nil, fmt.Errorf("tool %q: %w", t.Name, execErr)
	}
	return result, nil
}

// --- Approval error ----------------------------------------------------------

// ApprovalError is returned when a tool requires human approval before execution.
type ApprovalError struct {
	Tool string
	Args map[string]any
}

func (e *ApprovalError) Error() string {
	return fmt.Sprintf("tool %q requires approval", e.Tool)
}

// --- Registry ----------------------------------------------------------------

// Registry is a thread-safe collection of tools keyed by name.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]*Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]*Tool)}
}

// Register adds a tool to the registry. It panics if a tool with the same name exists.
func (r *Registry) Register(t *Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.tools[t.Name]; dup {
		panic("tool: duplicate registration of " + t.Name)
	}
	r.tools[t.Name] = t
}

// Get retrieves a tool by name, returning nil if not found.
func (r *Registry) Get(name string) *Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// List returns all registered tools.
func (r *Registry) List() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// --- Type checking -----------------------------------------------------------

func checkType(expected string, val any) error {
	if val == nil {
		return nil // nil is acceptable; required check happens separately
	}
	switch strings.ToLower(expected) {
	case "string":
		if _, ok := val.(string); !ok {
			return fmt.Errorf("expected string, got %T", val)
		}
	case "number":
		switch val.(type) {
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64:
			// ok
		default:
			return fmt.Errorf("expected number, got %T", val)
		}
	case "boolean":
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", val)
		}
	case "object":
		if _, ok := val.(map[string]any); !ok {
			return fmt.Errorf("expected object, got %T", val)
		}
	case "array":
		if _, ok := val.([]any); !ok {
			return fmt.Errorf("expected array, got %T", val)
		}
	default:
		// Unknown type string — skip validation
	}
	return nil
}
