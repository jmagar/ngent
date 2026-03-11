package codex

import (
	"strings"
	"testing"
)

func TestParseSessionTranscript(t *testing.T) {
	raw := strings.Join([]string{
		`{"timestamp":"2026-03-11T10:03:50.218Z","type":"response_item","payload":{"type":"message","role":"developer","content":[{"type":"input_text","text":"ignore"}]}}`,
		`{"timestamp":"2026-03-11T10:03:50.219Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}}`,
		`{"timestamp":"2026-03-11T10:03:50.220Z","type":"event_msg","payload":{"type":"user_message","message":"ignored duplicate"}}`,
		`{"timestamp":"2026-03-11T10:03:51.000Z","type":"response_item","payload":{"type":"message","role":"assistant","phase":"commentary","content":[{"type":"output_text","text":"working"}]}}`,
		`{"timestamp":"2026-03-11T10:03:51.906Z","type":"response_item","payload":{"type":"message","role":"assistant","phase":"final_answer","content":[{"type":"output_text","text":"world"}]}}`,
	}, "\n")

	messages, err := parseSessionTranscript(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("parseSessionTranscript(): %v", err)
	}
	if got, want := len(messages), 2; got != want {
		t.Fatalf("len(messages) = %d, want %d", got, want)
	}
	if got := messages[0].Role; got != "user" {
		t.Fatalf("messages[0].Role = %q, want %q", got, "user")
	}
	if got := messages[0].Content; got != "hello" {
		t.Fatalf("messages[0].Content = %q, want %q", got, "hello")
	}
	if got := messages[1].Role; got != "assistant" {
		t.Fatalf("messages[1].Role = %q, want %q", got, "assistant")
	}
	if got := messages[1].Content; got != "world" {
		t.Fatalf("messages[1].Content = %q, want %q", got, "world")
	}
}
