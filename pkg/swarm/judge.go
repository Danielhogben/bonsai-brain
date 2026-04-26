package swarm

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/donn/bonsai-brain/pkg/engine"
	"github.com/donn/bonsai-brain/pkg/openai"
)

// Judge scores TaskResults using an LLM judge model.
type Judge struct {
	Client *openai.Client
	Model  string // judge model name (e.g. "gpt-4o-mini")
}

// NewJudge creates a judge using the given OpenAI-compatible endpoint.
func NewJudge(baseURL, apiKey, model string) *Judge {
	return &Judge{
		Client: openai.NewClient(baseURL, apiKey, model),
		Model:  model,
	}
}

// ScorePrompt builds the system prompt for the judge.
func (j *Judge) ScorePrompt(task string, results []TaskResult) string {
	var b strings.Builder
	b.WriteString("You are an expert evaluator. Score each response to the task below.\n")
	b.WriteString("Rate each response from 1-10 on: Accuracy, Completeness, Clarity, Relevance.\n")
	b.WriteString("Respond in this exact format for each result:\n")
	b.WriteString("RESULT <index>: <score> | <one-sentence rationale>\n\n")
	b.WriteString("Task: ")
	b.WriteString(task)
	b.WriteString("\n\nResponses:\n")
	for i, r := range results {
		if r.Error != nil {
			fmt.Fprintf(&b, "[%d] ERROR: %v\n", i, r.Error)
			continue
		}
		out := r.Output
		if len(out) > 800 {
			out = out[:800] + "..."
		}
		fmt.Fprintf(&b, "[%d] %s\n", i, out)
	}
	return b.String()
}

// PickBest sends results to the judge and returns the index of the best result.
func (j *Judge) PickBest(ctx context.Context, task string, results []TaskResult) (int, error) {
	if len(results) == 0 {
		return -1, fmt.Errorf("no results to judge")
	}

	prompt := j.ScorePrompt(task, results)
	msg := engine.Message{Role: "user", Content: prompt}
	resp, err := j.Client.Stream(ctx, []engine.Message{msg}, nil)
	if err != nil {
		return -1, fmt.Errorf("judge request failed: %w", err)
	}

	bestIdx, bestScore := -1, -1
	for _, line := range strings.Split(resp.Content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "RESULT") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		idxStr := strings.Fields(parts[0])
		if len(idxStr) < 2 {
			continue
		}
		idx, err := strconv.Atoi(idxStr[1])
		if err != nil || idx < 0 || idx >= len(results) {
			continue
		}
		scoreStr := strings.Fields(parts[1])
		if len(scoreStr) == 0 {
			continue
		}
		score, err := strconv.Atoi(scoreStr[0])
		if err != nil {
			continue
		}
		if score > bestScore {
			bestScore = score
			bestIdx = idx
		}
	}

	if bestIdx == -1 {
		return 0, fmt.Errorf("judge returned no scores, falling back to first result")
	}
	return bestIdx, nil
}
