package swarm

import "testing"

func TestDashboardBuildStatus(t *testing.T) {
	reg := NewProviderRegistry([]ProviderConfig{})
	sw := NewSwarm(reg)
	d := NewDashboardServer(sw, reg)

	status := d.buildStatus()
	if status["type"] != "status" {
		t.Errorf("type = %q, want status", status["type"])
	}
	if status["agent_count"] != 0 {
		t.Errorf("agent_count = %v, want 0", status["agent_count"])
	}
	if _, ok := status["timestamp"]; !ok {
		t.Error("missing timestamp")
	}
}

func TestDashboardBroadcastNoClients(t *testing.T) {
	reg := NewProviderRegistry([]ProviderConfig{})
	sw := NewSwarm(reg)
	d := NewDashboardServer(sw, reg)
	// Should not panic with no clients
	d.BroadcastStatus()
}
