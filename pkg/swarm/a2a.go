// Package swarm implements agent-to-agent (A2A) protocol for inter-agent communication.
//
// The A2A protocol enables agents in a swarm to:
// - Discover each other's capabilities
// - Delegate tasks to specialized agents
// - Share results and coordinate workflows
package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// A2AMessage represents a message exchanged between agents.
type A2AMessage struct {
	ID        string          `json:"id"`
	From      string          `json:"from"`
	To        string          `json:"to"`
	Type      A2AMessageType  `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

// A2AMessageType defines the type of A2A message.
type A2AMessageType string

const (
	A2ARequest  A2AMessageType = "request"  // Task request from one agent to another
	A2AResponse A2AMessageType = "response" // Task response
	A2AProbe    A2AMessageType = "probe"    // Capability probe
	A2AAnnounce A2AMessageType = "announce" // Agent announcement
	A2ACancel   A2AMessageType = "cancel"   // Cancel a pending task
)

// A2ATaskRequest is the payload for a task request.
type A2ATaskRequest struct {
	TaskID     string            `json:"task_id"`
	Prompt     string            `json:"prompt"`
	System     string            `json:"system,omitempty"`
	MaxTokens  int               `json:"max_tokens,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// A2ATaskResponse is the payload for a task response.
type A2ATaskResponse struct {
	TaskID  string `json:"task_id"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
	Latency int64  `json:"latency_ms"`
}

// A2AProbeRequest asks an agent about its capabilities.
type A2AProbeRequest struct{}

// A2AProbeResponse describes an agent's capabilities.
type A2AProbeResponse struct {
	AgentID    string   `json:"agent_id"`
	Model      string   `json:"model"`
	Provider   string   `json:"provider"`
	Capabilities []string `json:"capabilities"`
	MaxIter    int      `json:"max_iter"`
}

// A2AHandler processes incoming A2A messages and returns a response.
type A2AHandler func(ctx context.Context, msg *A2AMessage) (*A2AMessage, error)

// A2ABus is a message bus for agent-to-agent communication.
type A2ABus struct {
	mu       sync.RWMutex
	agents   map[string]*A2AAgent // registered agents
	handlers map[A2AMessageType]A2AHandler
	messageLog []A2AMessage
}

// A2AAgent represents an agent on the A2A bus.
type A2AAgent struct {
	ID           string
	Capabilities []string
	Bus          *A2ABus
	SendFunc     func(ctx context.Context, msg *A2AMessage) error
}

// NewA2ABus creates a new agent-to-agent message bus.
func NewA2ABus() *A2ABus {
	bus := &A2ABus{
		agents:   make(map[string]*A2AAgent),
		handlers: make(map[A2AMessageType]A2AHandler),
	}
	return bus
}

// RegisterAgent registers an agent with the bus.
func (b *A2ABus) RegisterAgent(id string, capabilities []string) *A2AAgent {
	b.mu.Lock()
	defer b.mu.Unlock()

	agent := &A2AAgent{
		ID:           id,
		Capabilities: capabilities,
		Bus:          b,
	}
	b.agents[id] = agent
	return agent
}

// UnregisterAgent removes an agent from the bus.
func (b *A2ABus) UnregisterAgent(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.agents, id)
}

// GetAgent returns an agent by ID.
func (b *A2ABus) GetAgent(id string) (*A2AAgent, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	agent, ok := b.agents[id]
	return agent, ok
}

// FindAgentsByCapability finds agents with a specific capability.
func (b *A2ABus) FindAgentsByCapability(capability string) []*A2AAgent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var matches []*A2AAgent
	for _, agent := range b.agents {
		for _, cap := range agent.Capabilities {
			if cap == capability {
				matches = append(matches, agent)
				break
			}
		}
	}
	return matches
}

// Send sends a message to a specific agent.
func (b *A2ABus) Send(ctx context.Context, msg *A2AMessage) error {
	b.mu.RLock()
	agent, ok := b.agents[msg.To]
	b.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent %s not found", msg.To)
	}

	// Log the message
	b.mu.Lock()
	b.messageLog = append(b.messageLog, *msg)
	b.mu.Unlock()

	// Route to handler
	if handler, ok := b.handlers[msg.Type]; ok {
		resp, err := handler(ctx, msg)
		if err != nil {
			return err
		}
		if resp != nil {
			return b.Send(ctx, resp)
		}
	}

	// Default: invoke agent's send function
	if agent.SendFunc != nil {
		return agent.SendFunc(ctx, msg)
	}

	return nil
}

// Broadcast sends a message to all registered agents.
func (b *A2ABus) Broadcast(ctx context.Context, msg *A2AMessage) error {
	b.mu.RLock()
	agents := make([]*A2AAgent, 0, len(b.agents))
	for _, a := range b.agents {
		agents = append(agents, a)
	}
	b.mu.RUnlock()

	for _, agent := range agents {
		if agent.ID == msg.From {
			continue // don't send to self
		}
		m := *msg
		m.To = agent.ID
		if err := b.Send(ctx, &m); err != nil {
			// Log but continue broadcasting
			fmt.Printf("A2A broadcast to %s failed: %v\n", agent.ID, err)
		}
	}
	return nil
}

// RegisterHandler registers a handler for a specific message type.
func (b *A2ABus) RegisterHandler(msgType A2AMessageType, handler A2AHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[msgType] = handler
}

// GetMessageLog returns the message log.
func (b *A2ABus) GetMessageLog() []A2AMessage {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]A2AMessage, len(b.messageLog))
	copy(result, b.messageLog)
	return result
}

// AgentCount returns the number of registered agents.
func (b *A2ABus) AgentCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.agents)
}

// NewA2AMessage creates a new A2A message with timestamp.
func NewA2AMessage(from, to string, msgType A2AMessageType, payload interface{}) (*A2AMessage, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &A2AMessage{
		ID:        fmt.Sprintf("a2a-%d", time.Now().UnixNano()),
		From:      from,
		To:        to,
		Type:      msgType,
		Payload:   b,
		Timestamp: time.Now().UTC(),
	}, nil
}

// SwarmA2A integrates A2A protocol with the Swarm.
type SwarmA2A struct {
	Bus  *A2ABus
	swarm *Swarm
}

// NewSwarmA2A creates an A2A-enabled swarm.
func NewSwarmA2A(swarm *Swarm) *SwarmA2A {
	sa := &SwarmA2A{
		Bus:   NewA2ABus(),
		swarm: swarm,
	}
	sa.setupDefaultHandlers()
	return sa
}

// setupDefaultHandlers sets up default A2A message handlers.
func (sa *SwarmA2A) setupDefaultHandlers() {
	// Handle capability probes
	sa.Bus.RegisterHandler(A2AProbe, func(ctx context.Context, msg *A2AMessage) (*A2AMessage, error) {
		agent, ok := sa.Bus.GetAgent(msg.To)
		if !ok {
			return nil, fmt.Errorf("agent %s not found", msg.To)
		}
		resp := A2AProbeResponse{
			AgentID:    agent.ID,
			Capabilities: agent.Capabilities,
		}
		return NewA2AMessage(msg.To, msg.From, A2AProbe, resp)
	})

	// Handle task requests
	sa.Bus.RegisterHandler(A2ARequest, func(ctx context.Context, msg *A2AMessage) (*A2AMessage, error) {
		var req A2ATaskRequest
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			return nil, err
		}

		// Find the swarm agent and execute
		sa.swarm.mu.RLock()
		swarmAgent, ok := sa.swarm.Agents[msg.To]
		sa.swarm.mu.RUnlock()

		if !ok {
			return nil, fmt.Errorf("swarm agent %s not found", msg.To)
		}

		start := time.Now()
		_ = swarmAgent // Available for future integration
		latency := time.Since(start)

		resp := A2ATaskResponse{
			TaskID:  req.TaskID,
			Output:  "A2A task received: " + req.Prompt,
			Latency: latency.Milliseconds(),
		}

		return NewA2AMessage(msg.To, msg.From, A2AResponse, resp)
	})
}

// DiscoverCapabilities probes all agents for their capabilities.
func (sa *SwarmA2A) DiscoverCapabilities(ctx context.Context) map[string]*A2AProbeResponse {
	capabilities := make(map[string]*A2AProbeResponse)

	sa.swarm.mu.RLock()
	agents := make(map[string]*SwarmAgent)
	for k, v := range sa.swarm.Agents {
		agents[k] = v
	}
	sa.swarm.mu.RUnlock()

	for id := range agents {
		probe, _ := NewA2AMessage("discover", id, A2AProbe, A2AProbeRequest{})
		if probe != nil {
			_ = sa.Bus.Send(ctx, probe)
			// In production, the handler would populate capabilities
		}
	}

	return capabilities
}

// DelegateTask sends a task to a specific agent via A2A.
func (sa *SwarmA2A) DelegateTask(ctx context.Context, from, to string, task Task) (*A2ATaskResponse, error) {
	req := A2ATaskRequest{
		TaskID:    task.ID,
		Prompt:    task.Prompt,
		System:    task.System,
		MaxTokens: task.MaxIter,
	}

	msg, err := NewA2AMessage(from, to, A2ARequest, req)
	if err != nil {
		return nil, err
	}

	sa.Bus.Send(ctx, msg)

	// In production, this would wait for and parse the response
	return &A2ATaskResponse{
		TaskID: req.TaskID,
		Output: "Task delegated via A2A",
	}, nil
}
