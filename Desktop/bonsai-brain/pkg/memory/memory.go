// Package memory provides conversation memory management for Bonsai Brain agents.
// It tracks per-session message history, estimates token usage, and automatically
// compresses old turns via summarization when thresholds are exceeded.
package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

// Config controls memory behaviour.
type Config struct {
	// MaxMessages before summarization triggers. 0 = unlimited.
	MaxMessages int
	// MaxTokens (estimated) before summarization triggers. 0 = unlimited.
	MaxTokens int
	// SummaryModel is a lightweight model client used to generate summaries.
	// If nil, old messages are dropped instead of summarized.
	SummaryModel Summarizer
	// KeepSystemPrompt ensures the original system prompt is never summarized away.
	KeepSystemPrompt bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxMessages:      20,
		MaxTokens:        4000,
		KeepSystemPrompt: true,
	}
}

// Summarizer is the minimal interface required to generate a summary string.
type Summarizer interface {
	Summarize(ctx context.Context, text string) (string, error)
}

// ---------------------------------------------------------------------------
// Turn
// ---------------------------------------------------------------------------

// Turn represents a single user ↔ assistant exchange.
type Turn struct {
	User      string
	Assistant string
	ToolCalls []ToolCallRecord
	Timestamp time.Time
}

// ToolCallRecord captures a tool invocation and its result.
type ToolCallRecord struct {
	Name   string
	Args   map[string]any
	Result string
	Error  string
}

// ---------------------------------------------------------------------------
// Memory
// ---------------------------------------------------------------------------

// Memory holds conversation state for a single session.
type Memory struct {
	mu     sync.RWMutex
	config Config

	SystemPrompt string
	Messages     []Message // raw messages after system prompt
	Turns        []Turn    // structured turns for summary rebuild
	Summary      string    // compressed history of old turns
}

// Message is a lightweight representation for the engine.
type Message struct {
	Role    string
	Content string
}

// New creates an empty Memory.
func New(cfg Config) *Memory {
	return &Memory{config: cfg}
}

// SetSystemPrompt stores the system prompt. It is protected from summarization
// when KeepSystemPrompt is true.
func (m *Memory) SetSystemPrompt(prompt string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SystemPrompt = prompt
}

// AddTurn appends a user/assistant exchange and prunes if over budget.
func (m *Memory) AddTurn(ctx context.Context, turn Turn) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	turn.Timestamp = time.Now().UTC()
	m.Turns = append(m.Turns, turn)

	// Append raw messages.
	m.Messages = append(m.Messages, Message{Role: "user", Content: turn.User})
	if turn.Assistant != "" {
		m.Messages = append(m.Messages, Message{Role: "assistant", Content: turn.Assistant})
	}
	for _, tc := range turn.ToolCalls {
		content := fmt.Sprintf("tool %s(%v) → %s", tc.Name, tc.Args, tc.Result)
		if tc.Error != "" {
			content = fmt.Sprintf("tool %s(%v) → error: %s", tc.Name, tc.Args, tc.Error)
		}
		m.Messages = append(m.Messages, Message{Role: "tool", Content: content})
	}

	return m.maybeCompress(ctx)
}

// BuildMessages returns the full message list for the engine:
// [system prompt, summary, recent messages].
func (m *Memory) BuildMessages() []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []Message
	if m.SystemPrompt != "" {
		out = append(out, Message{Role: "system", Content: m.SystemPrompt})
	}
	if m.Summary != "" {
		out = append(out, Message{Role: "system", Content: "Previous conversation summary: " + m.Summary})
	}
	out = append(out, m.Messages...)
	return out
}

// Clear wipes all state except the system prompt.
func (m *Memory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = nil
	m.Turns = nil
	m.Summary = ""
}

// TokenEstimate returns a rough word-count-based token estimate.
func (m *Memory) TokenEstimate() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tokenEstimateUnlocked()
}

func (m *Memory) tokenEstimateUnlocked() int {
	total := len(m.SystemPrompt) / 4
	total += len(m.Summary) / 4
	for _, msg := range m.Messages {
		total += len(msg.Content) / 4
	}
	return total
}

// ---------------------------------------------------------------------------
// Compression
// ---------------------------------------------------------------------------

func (m *Memory) maybeCompress(ctx context.Context) error {
	overMessages := m.config.MaxMessages > 0 && len(m.Messages) > m.config.MaxMessages
	overTokens := m.config.MaxTokens > 0 && m.tokenEstimateUnlocked() > m.config.MaxTokens
	if !overMessages && !overTokens {
		return nil
	}

	// Decide how many turns to compress (half of them).
	compressCount := len(m.Turns) / 2
	if compressCount < 1 {
		compressCount = 1
	}

	oldTurns := m.Turns[:compressCount]
	m.Turns = m.Turns[compressCount:]

	// Rebuild Messages from remaining turns.
	m.Messages = nil
	for _, turn := range m.Turns {
		m.Messages = append(m.Messages, Message{Role: "user", Content: turn.User})
		if turn.Assistant != "" {
			m.Messages = append(m.Messages, Message{Role: "assistant", Content: turn.Assistant})
		}
		for _, tc := range turn.ToolCalls {
			content := fmt.Sprintf("tool %s(%v) → %s", tc.Name, tc.Args, tc.Result)
			if tc.Error != "" {
				content = fmt.Sprintf("tool %s(%v) → error: %s", tc.Name, tc.Args, tc.Error)
			}
			m.Messages = append(m.Messages, Message{Role: "tool", Content: content})
		}
	}

	// Summarize or drop old turns.
	if m.config.SummaryModel != nil {
		text := turnsToText(oldTurns)
		summary, err := m.config.SummaryModel.Summarize(ctx, text)
		if err == nil && summary != "" {
			if m.Summary != "" {
				m.Summary = m.Summary + "\n" + summary
			} else {
				m.Summary = summary
			}
		}
	}
	return nil
}

func turnsToText(turns []Turn) string {
	var parts []string
	for _, t := range turns {
		parts = append(parts, fmt.Sprintf("User: %s\nAssistant: %s", t.User, t.Assistant))
	}
	return strings.Join(parts, "\n---\n")
}
