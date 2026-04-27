package swarm

import (
	
	"testing"
)

func TestMCPServerCreation(t *testing.T) {
	server := NewMCPServer("test-server", "http://localhost:8080", MCPTransportHTTP)
	if server == nil {
		t.Fatal("NewMCPServer returned nil")
	}
	if server.Name != "test-server" {
		t.Errorf("expected name 'test-server', got '%s'", server.Name)
	}
	if server.URL != "http://localhost:8080" {
		t.Errorf("expected URL 'http://localhost:8080', got '%s'", server.URL)
	}
	if server.IsConnected() {
		t.Error("expected not connected initially")
	}
}

func TestMCPServerCapabilities(t *testing.T) {
	server := NewMCPServer("test-server", "http://localhost:8080", MCPTransportHTTP)
	if len(server.Capabilities) != 0 {
		t.Errorf("expected 0 capabilities initially, got %d", len(server.Capabilities))
	}
}

func TestMCPServerTools(t *testing.T) {
	server := NewMCPServer("test-server", "http://localhost:8080", MCPTransportHTTP)
	tools := server.GetTools()
	if len(tools) != 0 {
		t.Errorf("expected 0 tools initially, got %d", len(tools))
	}
}

func TestMCPManagerCreation(t *testing.T) {
	manager := NewMCPManager()
	if manager == nil {
		t.Fatal("NewMCPManager returned nil")
	}
}

func TestMCPManagerAddRemoveServer(t *testing.T) {
	manager := NewMCPManager()
	server := NewMCPServer("server-1", "http://localhost:8080", MCPTransportHTTP)

	manager.AddServer(server)
	got, ok := manager.GetServer("server-1")
	if !ok {
		t.Fatal("GetServer returned false after AddServer")
	}
	if got.Name != "server-1" {
		t.Errorf("expected server name 'server-1', got '%s'", got.Name)
	}

	manager.RemoveServer("server-1")
	_, ok = manager.GetServer("server-1")
	if ok {
		t.Error("GetServer returned true after RemoveServer")
	}
}

func TestMCPManagerGetAllTools(t *testing.T) {
	manager := NewMCPManager()
	tools := manager.GetAllTools()
	if len(tools) != 0 {
		t.Errorf("expected 0 tools from empty manager, got %d", len(tools))
	}
}

func TestMCPManagerFindToolByName(t *testing.T) {
	manager := NewMCPManager()
	server, tool := manager.FindToolByName("nonexistent")
	if server != nil {
		t.Error("expected nil server for nonexistent tool")
	}
	if tool != nil {
		t.Error("expected nil tool for nonexistent tool")
	}
}

func TestMCPToolTypes(t *testing.T) {
	tool := MCPTool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	if tool.Name != "test-tool" {
		t.Errorf("expected tool name 'test-tool', got '%s'", tool.Name)
	}
}

func TestMCPResourceTypes(t *testing.T) {
	resource := MCPResource{
		URI:         "file:///test.txt",
		Name:        "test.txt",
		Description: "A test file",
		MimeType:    "text/plain",
	}

	if resource.URI != "file:///test.txt" {
		t.Errorf("expected URI 'file:///test.txt', got '%s'", resource.URI)
	}
}

func TestMCPMessageTypes(t *testing.T) {
	tests := []struct {
		name     string
		msgType  A2AMessageType
		expected string
	}{
		{"request", A2ARequest, "request"},
		{"response", A2AResponse, "response"},
		{"probe", A2AProbe, "probe"},
		{"announce", A2AAnnounce, "announce"},
		{"cancel", A2ACancel, "cancel"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.msgType) != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, string(tt.msgType))
			}
		})
	}
}

func TestA2AMessageTypes(t *testing.T) {
	msg := A2AMessage{
		ID:   "test-id",
		From: "agent-1",
		To:   "agent-2",
		Type: A2ARequest,
	}

	if msg.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", msg.ID)
	}
	if msg.From != "agent-1" {
		t.Errorf("expected From 'agent-1', got '%s'", msg.From)
	}
	if msg.To != "agent-2" {
		t.Errorf("expected To 'agent-2', got '%s'", msg.To)
	}
}
