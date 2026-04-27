package swarm

import (
	"context"
	"encoding/json"
	"testing"
)

func TestA2ABusCreation(t *testing.T) {
	bus := NewA2ABus()
	if bus == nil {
		t.Fatal("NewA2ABus returned nil")
	}
	if bus.AgentCount() != 0 {
		t.Errorf("expected 0 agents, got %d", bus.AgentCount())
	}
}

func TestA2ABusRegisterAgent(t *testing.T) {
	bus := NewA2ABus()
	agent := bus.RegisterAgent("agent-1", []string{"code", "research"})

	if agent == nil {
		t.Fatal("RegisterAgent returned nil")
	}
	if agent.ID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got '%s'", agent.ID)
	}
	if bus.AgentCount() != 1 {
		t.Errorf("expected 1 agent, got %d", bus.AgentCount())
	}
}

func TestA2ABusFindAgentsByCapability(t *testing.T) {
	bus := NewA2ABus()
	bus.RegisterAgent("coder", []string{"code", "debug"})
	bus.RegisterAgent("writer", []string{"creative", "writing"})
	bus.RegisterAgent("researcher", []string{"research", "analysis"})

	codeAgents := bus.FindAgentsByCapability("code")
	if len(codeAgents) != 1 {
		t.Errorf("expected 1 code agent, got %d", len(codeAgents))
	}
	if codeAgents[0].ID != "coder" {
		t.Errorf("expected coder, got %s", codeAgents[0].ID)
	}

	researchAgents := bus.FindAgentsByCapability("research")
	if len(researchAgents) != 1 {
		t.Errorf("expected 1 research agent, got %d", len(researchAgents))
	}
}

func TestA2ABusUnregisterAgent(t *testing.T) {
	bus := NewA2ABus()
	bus.RegisterAgent("agent-1", []string{"code"})
	bus.UnregisterAgent("agent-1")

	if bus.AgentCount() != 0 {
		t.Errorf("expected 0 agents after unregister, got %d", bus.AgentCount())
	}
}

func TestNewA2AMessage(t *testing.T) {
	payload := A2ATaskRequest{
		TaskID: "task-1",
		Prompt: "Hello",
	}
	msg, err := NewA2AMessage("agent-1", "agent-2", A2ARequest, payload)
	if err != nil {
		t.Fatalf("NewA2AMessage failed: %v", err)
	}
	if msg.From != "agent-1" {
		t.Errorf("expected from 'agent-1', got '%s'", msg.From)
	}
	if msg.To != "agent-2" {
		t.Errorf("expected to 'agent-2', got '%s'", msg.To)
	}
	if msg.Type != A2ARequest {
		t.Errorf("expected type A2ARequest, got '%s'", msg.Type)
	}
	if msg.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestA2ABusSendWithHandler(t *testing.T) {
	bus := NewA2ABus()
	bus.RegisterAgent("agent-1", []string{"code"})
	bus.RegisterAgent("agent-2", []string{"research"})

	received := false
	bus.RegisterHandler(A2ARequest, func(ctx context.Context, msg *A2AMessage) (*A2AMessage, error) {
		received = true
		var req A2ATaskRequest
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			return nil, err
		}
		return nil, nil
	})

	payload := A2ATaskRequest{
		TaskID: "task-1",
		Prompt: "Test prompt",
	}
	msg, _ := NewA2AMessage("agent-1", "agent-2", A2ARequest, payload)

	err := bus.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if !received {
		t.Error("handler was not called")
	}
}

func TestA2ABusGetMessageLog(t *testing.T) {
	bus := NewA2ABus()
	bus.RegisterAgent("agent-1", []string{"code"})
	bus.RegisterAgent("agent-2", []string{"research"})

	bus.RegisterHandler(A2ARequest, func(ctx context.Context, msg *A2AMessage) (*A2AMessage, error) {
		return nil, nil
	})

	payload := A2ATaskRequest{TaskID: "task-1", Prompt: "Test"}
	msg, _ := NewA2AMessage("agent-1", "agent-2", A2ARequest, payload)
	_ = bus.Send(context.Background(), msg)

	log := bus.GetMessageLog()
	if len(log) != 1 {
		t.Errorf("expected 1 message in log, got %d", len(log))
	}
}

func TestSwarmA2AIntegration(t *testing.T) {
	// Create a mock swarm with provider registry
	configs := []ProviderConfig{
		{
			Type:      ProviderLocal,
			BaseURL:   "http://localhost:11434",
			Models:    []string{"test-model"},
			RateLimit: 10,
		},
	}
	registry := NewProviderRegistry(configs)
	swarm := NewSwarm(registry)

	sa := NewSwarmA2A(swarm)

	if sa.Bus == nil {
		t.Fatal("SwarmA2A.Bus is nil")
	}
	if sa.swarm == nil {
		t.Fatal("SwarmA2A.swarm is nil")
	}
}
