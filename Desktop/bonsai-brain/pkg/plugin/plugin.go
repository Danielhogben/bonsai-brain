// Package plugin provides a 4-component plugin system inspired by elizaOS.
// A Plugin bundles Actions (things the agent can do), Providers (context
// injectors), Evaluators (post-response classifiers), and Services
// (long-lived background workers).  The Manager is a thread-safe registry
// that composes plugins at runtime.
package plugin

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/donn/bonsai-brain/pkg/engine"
)

// ---------------------------------------------------------------------------
// Shared types
// ---------------------------------------------------------------------------

// AgentRuntime is the minimal surface a plugin component sees when it
// executes.  Concrete agent implementations embed or wrap this.
type AgentRuntime struct {
	Ctx     context.Context
	Engine  *engine.QueryEngine
	Message engine.Message  // the user message being processed
	State   map[string]any   // mutable per-turn state bag
}

// ---------------------------------------------------------------------------
// Action
// ---------------------------------------------------------------------------

// Action represents something the agent can *do* in response to a message
// (e.g. call a tool, send a notification, write a file).
type Action interface {
	Name() string
	Validate(runtime *AgentRuntime) bool
	Handle(runtime *AgentRuntime, opts map[string]any) (string, error)
}

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

// ProviderResult bundles the three things a provider can return.
type ProviderResult struct {
	Text   string         // human-readable text injected into the prompt
	Values map[string]any // structured values other components can inspect
	Data   map[string]any // arbitrary payload (embeddings, blobs, refs …)
}

// Provider injects contextual information into the agent's prompt.
type Provider interface {
	Name() string
	Get(runtime *AgentRuntime) (ProviderResult, error)
}

// ---------------------------------------------------------------------------
// Evaluator
// ---------------------------------------------------------------------------

// Evaluator inspects the agent's state after a response and optionally
// triggers side-effects (memory writes, notifications, etc.).
type Evaluator interface {
	Name() string
	Validate(runtime *AgentRuntime) bool
	Handle(runtime *AgentRuntime) error
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// Service is a long-lived background worker managed by the plugin system.
type Service interface {
	Initialize(runtime *AgentRuntime) error
	Stop() error
}

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

// Plugin bundles the four component types under a single name.
type Plugin struct {
	Name        string
	Description string
	Actions     []Action
	Providers   []Provider
	Evaluators  []Evaluator
	Services    []Service
}

// ---------------------------------------------------------------------------
// Manager
// ---------------------------------------------------------------------------

// Manager is a thread-safe registry that holds all loaded plugins and
// exposes fast lookup tables for individual components.
type Manager struct {
	mu         sync.RWMutex
	plugins    map[string]*Plugin
	actions    map[string]Action
	providers  map[string]Provider
	evaluators map[string]Evaluator
	services   map[string]Service
}

// NewManager creates an empty plugin manager.
func NewManager() *Manager {
	return &Manager{
		plugins:    make(map[string]*Plugin),
		actions:    make(map[string]Action),
		providers:  make(map[string]Provider),
		evaluators: make(map[string]Evaluator),
		services:   make(map[string]Service),
	}
}

// Register adds a plugin to the manager.  It panics on duplicate names.
func (m *Manager) Register(p *Plugin) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, dup := m.plugins[p.Name]; dup {
		panic("plugin: duplicate registration of " + p.Name)
	}
	m.plugins[p.Name] = p

	for _, a := range p.Actions {
		m.actions[a.Name()] = a
	}
	for _, p := range p.Providers {
		m.providers[p.Name()] = p
	}
	for _, e := range p.Evaluators {
		m.evaluators[e.Name()] = e
	}
	for _, s := range p.Services {
		m.services[p.Name] = s // keyed by plugin name; one service per plugin
	}
}

// Plugin returns a registered plugin by name, or nil.
func (m *Manager) Plugin(name string) *Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.plugins[name]
}

// Plugins returns all registered plugins.
func (m *Manager) Plugins() []*Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		out = append(out, p)
	}
	return out
}

// Action returns a registered action by name, or nil.
func (m *Manager) Action(name string) Action {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.actions[name]
}

// Actions returns all registered actions.
func (m *Manager) Actions() []Action {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Action, 0, len(m.actions))
	for _, a := range m.actions {
		out = append(out, a)
	}
	return out
}

// MatchingActions returns actions whose Validate returns true for the given runtime.
func (m *Manager) MatchingActions(runtime *AgentRuntime) []Action {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Action
	for _, a := range m.actions {
		if a.Validate(runtime) {
			out = append(out, a)
		}
	}
	return out
}

// Providers returns all registered providers.
func (m *Manager) Providers() []Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Provider, 0, len(m.providers))
	for _, p := range m.providers {
		out = append(out, p)
	}
	return out
}

// Evaluators returns all registered evaluators.
func (m *Manager) Evaluators() []Evaluator {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Evaluator, 0, len(m.evaluators))
	for _, e := range m.evaluators {
		out = append(out, e)
	}
	return out
}

// MatchingEvaluators returns evaluators whose Validate returns true.
func (m *Manager) MatchingEvaluators(runtime *AgentRuntime) []Evaluator {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Evaluator
	for _, e := range m.evaluators {
		if e.Validate(runtime) {
			out = append(out, e)
		}
	}
	return out
}

// Services returns all registered services.
func (m *Manager) Services() []Service {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Service, 0, len(m.services))
	for _, s := range m.services {
		out = append(out, s)
	}
	return out
}

// ---------------------------------------------------------------------------
// ComposeProviders
// ---------------------------------------------------------------------------

// ComposeProviders iterates the given providers, collects their results, and
// returns the concatenated text along with merged value/data maps.  Errors
// from individual providers are collected but do not halt composition.
func ComposeProviders(runtime *AgentRuntime, providers []Provider) (ProviderResult, []error) {
	var texts []string
	values := make(map[string]any)
	data := make(map[string]any)
	var errs []error

	for _, p := range providers {
		result, err := p.Get(runtime)
		if err != nil {
			errs = append(errs, fmt.Errorf("provider %q: %w", p.Name(), err))
			continue
		}
		if result.Text != "" {
			texts = append(texts, result.Text)
		}
		for k, v := range result.Values {
			values[k] = v
		}
		for k, v := range result.Data {
			data[k] = v
		}
	}

	return ProviderResult{
		Text:   strings.Join(texts, "\n\n"),
		Values: values,
		Data:   data,
	}, errs
}
