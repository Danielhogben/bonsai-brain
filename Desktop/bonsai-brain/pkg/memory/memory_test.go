package memory

import (
	"context"
	"strings"
	"testing"
)

type mockSummarizer struct{}

func (m *mockSummarizer) Summarize(_ context.Context, text string) (string, error) {
	return "SUMMARY: " + strings.Split(text, "\n")[0], nil
}

func TestMemory_AddTurnAndBuild(t *testing.T) {
	mem := New(DefaultConfig())
	mem.SetSystemPrompt("You are a test assistant.")

	ctx := context.Background()
	mem.AddTurn(ctx, Turn{User: "Hello", Assistant: "Hi there!"})
	mem.AddTurn(ctx, Turn{User: "What's 2+2?", Assistant: "4"})

	msgs := mem.BuildMessages()
	if len(msgs) != 5 { // system + 2 user + 2 assistant
		t.Fatalf("expected 5 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Fatalf("expected first message to be system, got %s", msgs[0].Role)
	}
}

func TestMemory_GuardrailBlocks(t *testing.T) {
	mem := New(Config{MaxMessages: 4, SummaryModel: &mockSummarizer{}})
	mem.SetSystemPrompt("System prompt.")

	ctx := context.Background()
	for i := 0; i < 10; i++ {
		mem.AddTurn(ctx, Turn{User: "msg", Assistant: "reply"})
	}

	msgs := mem.BuildMessages()
	// After compression we should have system + summary + remaining messages.
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages after compression, got %d", len(msgs))
	}
	if !strings.Contains(msgs[1].Content, "SUMMARY") {
		t.Fatalf("expected summary message, got: %s", msgs[1].Content)
	}
}

func TestMemory_Clear(t *testing.T) {
	mem := New(DefaultConfig())
	mem.SetSystemPrompt("System.")
	mem.AddTurn(context.Background(), Turn{User: "hi", Assistant: "hello"})
	mem.Clear()

	msgs := mem.BuildMessages()
	if len(msgs) != 1 || msgs[0].Content != "System." {
		t.Fatalf("expected only system prompt after clear, got %v", msgs)
	}
}
