package agents

import (
	"encoding/json"
	"testing"
)

func TestACPTranscriptCollector_ReplaysUserAndAssistantMessages(t *testing.T) {
	t.Parallel()

	collector := NewACPTranscriptCollector()
	updates := []json.RawMessage{
		json.RawMessage(`{"update":{"sessionUpdate":"user_message_chunk","content":{"type":"text","text":"hello"}}}`),
		json.RawMessage(`{"update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"world"}}}`),
	}
	for _, update := range updates {
		if err := collector.HandleRawUpdate(update); err != nil {
			t.Fatalf("HandleRawUpdate() error = %v", err)
		}
	}

	result := collector.Result()
	if got, want := len(result.Messages), 2; got != want {
		t.Fatalf("len(messages) = %d, want %d", got, want)
	}
	if got := result.Messages[0]; got.Role != "user" || got.Content != "hello" {
		t.Fatalf("messages[0] = %+v, want user hello", got)
	}
	if got := result.Messages[1]; got.Role != "assistant" || got.Content != "world" {
		t.Fatalf("messages[1] = %+v, want assistant world", got)
	}
}

func TestACPTranscriptCollector_MergesChunksUntilBoundary(t *testing.T) {
	t.Parallel()

	collector := NewACPTranscriptCollector()
	updates := []ACPUpdate{
		{Type: ACPUpdateTypeUserMessageChunk, Role: "user", Delta: "hello "},
		{Type: ACPUpdateTypeUserMessageChunk, Role: "user", Delta: "world"},
		{Type: ACPUpdateTypePlan},
		{Type: ACPUpdateTypeAgentMessageChunk, Role: "assistant", Delta: "done"},
	}
	for _, update := range updates {
		collector.HandleUpdate(update)
	}

	result := collector.Result()
	if got, want := len(result.Messages), 2; got != want {
		t.Fatalf("len(messages) = %d, want %d", got, want)
	}
	if got := result.Messages[0].Content; got != "hello world" {
		t.Fatalf("messages[0].Content = %q, want %q", got, "hello world")
	}
	if got := result.Messages[1].Content; got != "done" {
		t.Fatalf("messages[1].Content = %q, want %q", got, "done")
	}
}

func TestACPTranscriptCollector_SplitsByMessageID(t *testing.T) {
	t.Parallel()

	collector := NewACPTranscriptCollector()
	updates := []ACPUpdate{
		{Type: ACPUpdateTypeAgentMessageChunk, Role: "assistant", MessageID: "m1", Delta: "first"},
		{Type: ACPUpdateTypeAgentMessageChunk, Role: "assistant", MessageID: "m2", Delta: "second"},
	}
	for _, update := range updates {
		collector.HandleUpdate(update)
	}

	result := collector.Result()
	if got, want := len(result.Messages), 2; got != want {
		t.Fatalf("len(messages) = %d, want %d", got, want)
	}
	if got := result.Messages[0].Content; got != "first" {
		t.Fatalf("messages[0].Content = %q, want %q", got, "first")
	}
	if got := result.Messages[1].Content; got != "second" {
		t.Fatalf("messages[1].Content = %q, want %q", got, "second")
	}
}
