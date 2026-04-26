package swarm

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// DashboardServer broadcasts swarm status over WebSocket and serves a minimal HTML UI.
type DashboardServer struct {
	mu       sync.RWMutex
	clients  map[*websocket.Conn]bool
	swarm    *Swarm
	registry *ProviderRegistry
	server   *http.Server
}

// NewDashboardServer creates a dashboard bound to a swarm instance.
func NewDashboardServer(sw *Swarm, reg *ProviderRegistry) *DashboardServer {
	return &DashboardServer{
		clients:  make(map[*websocket.Conn]bool),
		swarm:    sw,
		registry: reg,
	}
}

// Run starts the HTTP server on the given address.
func (d *DashboardServer) Run(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", d.handleIndex)
	mux.HandleFunc("/ws", d.handleWS)
	d.server = &http.Server{Addr: addr, Handler: mux}
	return d.server.ListenAndServe()
}

// Stop shuts down the server gracefully.
func (d *DashboardServer) Stop(ctx context.Context) error {
	return d.server.Shutdown(ctx)
}

func (d *DashboardServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(dashboardHTML))
}

func (d *DashboardServer) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	d.mu.Lock()
	d.clients[conn] = true
	d.mu.Unlock()

	defer func() {
		d.mu.Lock()
		delete(d.clients, conn)
		d.mu.Unlock()
	}()

	// Send initial snapshot
	d.broadcastSnapshot(conn)

	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// BroadcastStatus sends the current swarm status to all connected clients.
func (d *DashboardServer) BroadcastStatus() {
	d.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(d.clients))
	for c := range d.clients {
		clients = append(clients, c)
	}
	d.mu.RUnlock()

	status := d.buildStatus()
	for _, c := range clients {
		c.WriteJSON(status)
	}
}

func (d *DashboardServer) broadcastSnapshot(conn *websocket.Conn) {
	conn.WriteJSON(d.buildStatus())
}

func (d *DashboardServer) buildStatus() map[string]any {
	return map[string]any{
		"type":        "status",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"agent_count": d.swarm.AgentCount(),
		"providers":   len(d.registry.providers),
	}
}

const dashboardHTML = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Bonsai Brain Swarm Dashboard</title>
<style>
body { font-family: sans-serif; background: #0d1117; color: #c9d1d9; margin: 0; padding: 2rem; }
h1 { color: #58a6ff; }
#status { background: #161b22; padding: 1rem; border-radius: 8px; margin-bottom: 1rem; }
#log { background: #161b22; padding: 1rem; border-radius: 8px; height: 400px; overflow-y: auto; font-family: monospace; }
.entry { margin: 0.25rem 0; padding: 0.25rem 0; border-bottom: 1px solid #30363d; }
.connected { color: #3fb950; }
.disconnected { color: #f85149; }
</style>
</head>
<body>
<h1>🌳 Bonsai Brain Swarm Dashboard</h1>
<div id="status">Connecting...</div>
<div id="log"></div>
<script>
const ws = new WebSocket('ws://' + location.host + '/ws');
const statusEl = document.getElementById('status');
const logEl = document.getElementById('log');
ws.onopen = () => { statusEl.innerHTML = '<span class="connected">● Connected</span>'; };
ws.onclose = () => { statusEl.innerHTML = '<span class="disconnected">● Disconnected</span>'; };
ws.onmessage = (ev) => {
  const data = JSON.parse(ev.data);
  const div = document.createElement('div');
  div.className = 'entry';
  div.textContent = '[' + data.timestamp + '] Agents: ' + data.agent_count + ' | Providers: ' + data.providers;
  logEl.prepend(div);
};
</script>
</body>
</html>`
