package swarm

import (
	"fmt"
	"testing"
)

func TestJudgeScorePrompt(t *testing.T) {
	j := NewJudge("http://localhost:8787/v1", "sk-dummy", "openrouter/free")
	results := []TaskResult{
		{AgentID: "a1", Output: "Paris is the capital of France."},
		{AgentID: "a2", Output: "The capital of France is Paris, a city known for the Eiffel Tower."},
	}
	prompt := j.ScorePrompt("What is the capital of France?", results)
	if prompt == "" {
		t.Error("ScorePrompt returned empty string")
	}
	if !containsAny(prompt, "Paris", "capital", "France") {
		t.Error("ScorePrompt missing expected content")
	}
}

func TestBestQualityWinner(t *testing.T) {
	results := []TaskResult{
		{AgentID: "short", Output: "hi"},
		{AgentID: "long", Output: "hello world this is longer"},
		{AgentID: "err", Output: "", Error: fmt.Errorf("fail")},
	}
	res, err := BestQualityWinner(results)
	if err != nil {
		t.Fatalf("BestQualityWinner error: %v", err)
	}
	if res.Winner.AgentID != "long" {
		t.Errorf("winner = %q, want long", res.Winner.AgentID)
	}
}
