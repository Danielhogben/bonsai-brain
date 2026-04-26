// Package context provides a thread-safe registry of agent contexts inspired
// by agent-zero.  Each AgentContext carries an identity, a type, a free-form
// data map, and a creation timestamp.  The Registry supports CRUD operations
// and a goroutine-local "current" context via the standard library's
// context.Context propagation pattern.
package context

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ContextType classifies an agent context.
type ContextType string

const (
	ContextUser       ContextType = "user"
	ContextTask       ContextType = "task"
	ContextBackground ContextType = "background"
)

// AgentContext is a named, typed bag of data associated with an agent session.
type AgentContext struct {
	ID        string
	Name      string
	Type      ContextType
	Data      map[string]any
	CreatedAt time.Time
}

// New creates a new AgentContext with the given parameters.
func New(id, name string, ctxType ContextType) *AgentContext {
	return &AgentContext{
		ID:        id,
		Name:      name,
		Type:      ctxType,
		Data:      make(map[string]any),
		CreatedAt: time.Now().UTC(),
	}
}

// Set stores a key-value pair in the context's data map.
func (ac *AgentContext) Set(key string, value any) {
	ac.Data[key] = value
}

// Get retrieves a value from the context's data map.
func (ac *AgentContext) Get(key string) (any, bool) {
	v, ok := ac.Data[key]
	return v, ok
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

// Registry is a thread-safe store of agent contexts.
type Registry struct {
	mu       sync.RWMutex
	contexts map[string]*AgentContext
}

// NewRegistry creates an empty context registry.
func NewRegistry() *Registry {
	return &Registry{
		contexts: make(map[string]*AgentContext),
	}
}

// Create adds a new context to the registry.  It returns an error if the ID
// is already taken.
func (r *Registry) Create(ac *AgentContext) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.contexts[ac.ID]; exists {
		return fmt.Errorf("context: id %q already exists", ac.ID)
	}
	r.contexts[ac.ID] = ac
	return nil
}

// Get retrieves a context by ID.  Returns nil if not found.
func (r *Registry) Get(id string) *AgentContext {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.contexts[id]
}

// Remove deletes a context by ID.  Returns true if it existed.
func (r *Registry) Remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.contexts[id]
	if ok {
		delete(r.contexts, id)
	}
	return ok
}

// List returns all registered contexts.
func (r *Registry) List() []*AgentContext {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*AgentContext, 0, len(r.contexts))
	for _, ac := range r.contexts {
		out = append(out, ac)
	}
	return out
}

// ListByType returns contexts matching the given type.
func (r *Registry) ListByType(t ContextType) []*AgentContext {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*AgentContext
	for _, ac := range r.contexts {
		if ac.Type == t {
			out = append(out, ac)
		}
	}
	return out
}

// Count returns the number of registered contexts.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.contexts)
}

// ---------------------------------------------------------------------------
// Goroutine-local "current" context via context.Context
// ---------------------------------------------------------------------------

// contextKey is an unexported type used as the key for storing the current
// agent context in a standard context.Context.
type contextKey struct{}

// WithContext stores the current agent context in a context.Context.
func WithContext(ctx context.Context, ac *AgentContext) context.Context {
	return context.WithValue(ctx, contextKey{}, ac)
}

// FromContext retrieves the current agent context from a context.Context.
// Returns nil if no agent context is stored.
func FromContext(ctx context.Context) *AgentContext {
	ac, _ := ctx.Value(contextKey{}).(*AgentContext)
	return ac
}
