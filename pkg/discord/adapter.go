// Package discord provides a Discord channel adapter for Bonsai Brain.
// STUB: Full implementation restored from git history needs type unification
// between engine.Message and memory.Message before re-enabling.
package discord

// Adapter is a stub Discord adapter.
type Adapter struct{}

// NewAdapter creates a stub adapter.
func NewAdapter(token string) *Adapter {
	return &Adapter{}
}

// Start is a no-op stub.
func (a *Adapter) Start() error { return nil }

// Stop is a no-op stub.
func (a *Adapter) Stop() error { return nil }
