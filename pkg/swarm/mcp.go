// Package swarm implements MCP (Model Context Protocol) integration.
//
// MCP is an open protocol that standardizes how AI applications provide
// context to LLMs. It enables agents to discover and use tools from
// MCP servers.
package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// MCPVersion is the supported MCP protocol version.
const MCPVersion = "2024-11-05"

// MCPMessage represents a JSON-RPC message in the MCP protocol.
type MCPMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents a JSON-RPC error.
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCPTool represents a tool provided by an MCP server.
type MCPTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// MCPResource represents a resource provided by an MCP server.
type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// MCPServer represents a connection to an MCP server.
type MCPServer struct {
	mu          sync.RWMutex
	Name        string
	URL         string
	Transport   MCPTransport
	Tools       []MCPTool
	Resources   []MCPResource
	Capabilities map[string]bool
	client      *http.Client
	connected   bool
	sessionID   string
}

// MCPTransport defines the transport type for MCP communication.
type MCPTransport string

const (
	MCPTransportHTTP  MCPTransport = "http"
	MCPTransportStdio MCPTransport = "stdio"
)

// NewMCPServer creates a new MCP server connection.
func NewMCPServer(name, url string, transport MCPTransport) *MCPServer {
	return &MCPServer{
		Name:         name,
		URL:          url,
		Transport:    transport,
		Capabilities: make(map[string]bool),
		client:       &http.Client{Timeout: 30 * time.Second},
	}
}

// Connect initializes the MCP connection and negotiates capabilities.
func (s *MCPServer) Connect(ctx context.Context) error {
	// Send initialize request
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": MCPVersion,
			"capabilities": map[string]interface{}{
				"roots":    map[string]interface{}{"listChanged": true},
				"sampling": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "bonsai-brain",
				"version": "0.4.0",
			},
		},
	}

	resp, err := s.sendRequest(ctx, initReq)
	if err != nil {
		return fmt.Errorf("MCP initialize failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("MCP initialize error: %s", resp.Error.Message)
	}

	// Parse server capabilities
	if result, ok := resp.Result.(map[string]interface{}); ok {
		if caps, ok := result["capabilities"].(map[string]interface{}); ok {
			for k := range caps {
				s.Capabilities[k] = true
			}
		}
		if info, ok := result["serverInfo"].(map[string]interface{}); ok {
			if name, ok := info["name"].(string); ok {
				s.Name = name
			}
		}
	}

	// Send initialized notification
	notif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	_ = s.sendNotification(ctx, notif)

	s.connected = true
	return nil
}

// ListTools retrieves the list of tools from the MCP server.
func (s *MCPServer) ListTools(ctx context.Context) ([]MCPTool, error) {
	if !s.connected {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	req := MCPMessage{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}

	resp, err := s.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}

	var toolsResp struct {
		Tools []MCPTool `json:"tools"`
	}
	b, _ := json.Marshal(resp.Result)
	if err := json.Unmarshal(b, &toolsResp); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.Tools = toolsResp.Tools
	s.mu.Unlock()

	return toolsResp.Tools, nil
}

// CallTool invokes a tool on the MCP server.
func (s *MCPServer) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error) {
	if !s.connected {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	req := MCPMessage{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      name,
			"arguments": arguments,
		},
	}

	resp, err := s.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tools/call error: %s", resp.Error.Message)
	}

	return resp.Result, nil
}

// ListResources retrieves the list of resources from the MCP server.
func (s *MCPServer) ListResources(ctx context.Context) ([]MCPResource, error) {
	if !s.connected {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	req := MCPMessage{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "resources/list",
	}

	resp, err := s.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("resources/list error: %s", resp.Error.Message)
	}

	var resourcesResp struct {
		Resources []MCPResource `json:"resources"`
	}
	b, _ := json.Marshal(resp.Result)
	if err := json.Unmarshal(b, &resourcesResp); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.Resources = resourcesResp.Resources
	s.mu.Unlock()

	return resourcesResp.Resources, nil
}

// Disconnect closes the MCP connection.
func (s *MCPServer) Disconnect(ctx context.Context) error {
	// Send shutdown notification
	notif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "notifications/cancelled",
	}
	_ = s.sendNotification(ctx, notif)

	s.connected = false
	return nil
}

// IsConnected returns whether the server is connected.
func (s *MCPServer) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// GetTools returns cached tools.
func (s *MCPServer) GetTools() []MCPTool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]MCPTool, len(s.Tools))
	copy(result, s.Tools)
	return result
}

func (s *MCPServer) sendRequest(ctx context.Context, msg MCPMessage) (*MCPMessage, error) {
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.URL, io.NopCloser(
		&readerAdapter{b: b},
	))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if s.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", s.sessionID)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Capture session ID
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		s.sessionID = sid
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result MCPMessage
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *MCPServer) sendNotification(ctx context.Context, msg MCPMessage) error {
	_, err := s.sendRequest(ctx, msg)
	return err
}

// readerAdapter wraps a byte slice as an io.Reader.
type readerAdapter struct {
	b   []byte
	pos int
}

func (r *readerAdapter) Read(p []byte) (int, error) {
	if r.pos >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.pos:])
	r.pos += n
	return n, nil
}

// MCPManager manages multiple MCP server connections.
type MCPManager struct {
	mu      sync.RWMutex
	servers map[string]*MCPServer
}

// NewMCPManager creates a new MCP server manager.
func NewMCPManager() *MCPManager {
	return &MCPManager{
		servers: make(map[string]*MCPServer),
	}
}

// AddServer adds an MCP server to the manager.
func (m *MCPManager) AddServer(server *MCPServer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.servers[server.Name] = server
}

// RemoveServer removes an MCP server from the manager.
func (m *MCPManager) RemoveServer(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.servers, name)
}

// GetServer returns an MCP server by name.
func (m *MCPManager) GetServer(name string) (*MCPServer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	server, ok := m.servers[name]
	return server, ok
}

// ConnectAll connects to all registered MCP servers.
func (m *MCPManager) ConnectAll(ctx context.Context) error {
	m.mu.RLock()
	servers := make([]*MCPServer, 0, len(m.servers))
	for _, s := range m.servers {
		servers = append(servers, s)
	}
	m.mu.RUnlock()

	for _, server := range servers {
		if err := server.Connect(ctx); err != nil {
			fmt.Printf("Failed to connect to MCP server %s: %v\n", server.Name, err)
			continue
		}
		// Discover tools
		if _, err := server.ListTools(ctx); err != nil {
			fmt.Printf("Failed to list tools from %s: %v\n", server.Name, err)
		}
	}
	return nil
}

// DisconnectAll disconnects from all MCP servers.
func (m *MCPManager) DisconnectAll(ctx context.Context) error {
	m.mu.RLock()
	servers := make([]*MCPServer, 0, len(m.servers))
	for _, s := range m.servers {
		servers = append(servers, s)
	}
	m.mu.RUnlock()

	for _, server := range servers {
		_ = server.Disconnect(ctx)
	}
	return nil
}

// GetAllTools returns all tools from all connected MCP servers.
func (m *MCPManager) GetAllTools() map[string][]MCPTool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]MCPTool)
	for name, server := range m.servers {
		if server.IsConnected() {
			result[name] = server.GetTools()
		}
	}
	return result
}

// FindToolByName searches all servers for a tool by name.
func (m *MCPManager) FindToolByName(toolName string) (*MCPServer, *MCPTool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, server := range m.servers {
		if !server.IsConnected() {
			continue
		}
		for _, tool := range server.GetTools() {
			if tool.Name == toolName {
				return server, &tool
			}
		}
	}
	return nil, nil
}
