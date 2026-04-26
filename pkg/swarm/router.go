package swarm

import (
	"strings"
)

// Capability represents a task type an agent can handle.
type Capability string

const (
	CapCode      Capability = "code"
	CapCreative  Capability = "creative"
	CapResearch  Capability = "research"
	CapQuick     Capability = "quick"
	CapGeneral   Capability = "general"
)

// Router maps tasks to agents based on capability keywords.
type Router struct {
	agents []AgentConfig
}

// NewRouter creates a router for the given agent configs.
func NewRouter(agents []AgentConfig) *Router {
	return &Router{agents: agents}
}

// Route inspects the prompt and returns agents sorted by relevance.
func (r *Router) Route(prompt string) []AgentConfig {
	promptLower := strings.ToLower(prompt)
	scores := make(map[int]int)

	for i, ag := range r.agents {
		score := 0
		roleLower := strings.ToLower(ag.Role)
		toolsLower := strings.Join(ag.Tools, " ")
		toolsLower = strings.ToLower(toolsLower)

		// Keyword-based scoring (role-first to avoid tool overlap)
		switch {
		case containsAny(promptLower, "code", "program", "function", "bug", "fix", "refactor", "implement"):
			if containsAny(roleLower, "code", "dev", "engineer") {
				score += 10
			}
		case containsAny(promptLower, "write", "story", "poem", "creative", "blog", "essay", "draft"):
			if containsAny(roleLower, "creative", "writer", "poet") {
				score += 10
			}
		case containsAny(promptLower, "research", "search", "find", "look up", "investigate", "analyze"):
			if containsAny(roleLower, "research", "analyst") {
				score += 10
			}
		case containsAny(promptLower, "quick", "short", "brief", "one sentence", "summarize"):
			if ag.Model == "" || containsAny(ag.Model, "mini", "fast", "instant") {
				score += 10
			}
		}

		// Default fallback: all agents get a base score
		if score == 0 {
			score = 1
		}
		scores[i] = score
	}

	// Sort agents by score descending
	var sorted []AgentConfig
	for len(sorted) < len(r.agents) {
		bestIdx := -1
		bestScore := -1
		for i := range r.agents {
			if _, used := scores[i]; !used {
				continue
			}
			if scores[i] > bestScore {
				bestScore = scores[i]
				bestIdx = i
			}
		}
		if bestIdx == -1 {
			break
		}
		sorted = append(sorted, r.agents[bestIdx])
		delete(scores, bestIdx)
	}
	return sorted
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// containsWord checks for whole-word matches to avoid partial matches like
// "code-writer" matching "writer".
func containsWord(s string, word string) bool {
	// Simple whole-word check: ensure the character before and after the
	// match is not a letter.
	idx := strings.Index(s, word)
	if idx == -1 {
		return false
	}
	before := idx == 0 || !isLetter(s[idx-1])
	after := idx+len(word) == len(s) || !isLetter(s[idx+len(word)])
	return before && after
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
