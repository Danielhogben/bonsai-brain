package swarm

import "testing"

func TestRouterRoute(t *testing.T) {
	agents := []AgentConfig{
		{Name: "coder", Role: "developer", Model: "gpt-4o", Tools: []string{"filesystem"}},
		{Name: "creative", Role: "poet", Model: "claude-sonnet", Tools: []string{"web_search"}},
		{Name: "researcher", Role: "analyst", Model: "gpt-4o", Tools: []string{"web_search", "browser"}},
		{Name: "fast", Role: "responder", Model: "gpt-4o-mini", Tools: []string{}},
	}

	r := NewRouter(agents)

	// Code task should prioritize coder
	result := r.Route("Write a Go function to sort a slice")
	if result[0].Name != "coder" {
		t.Errorf("code task: first = %q, want coder", result[0].Name)
	}

	// Creative task should prioritize creative
	result = r.Route("Write a short poem about AI")
	if result[0].Name != "creative" {
		t.Errorf("creative task: first = %q, want creative", result[0].Name)
	}

	// Research task should prioritize researcher
	result = r.Route("Research the latest LLM benchmarks")
	if result[0].Name != "researcher" {
		t.Errorf("research task: first = %q, want researcher", result[0].Name)
	}

	// Quick task should prioritize fast agent
	result = r.Route("Give me a quick one-sentence summary")
	if result[0].Name != "fast" {
		t.Errorf("quick task: first = %q, want fast", result[0].Name)
	}
}
