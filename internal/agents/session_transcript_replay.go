package agents

import (
	"encoding/json"
	"strings"
)

// ACPTranscriptCollector rebuilds replayable user/assistant messages from ACP
// session/update notifications, including standard session/load replays.
type ACPTranscriptCollector struct {
	messages         []SessionTranscriptMessage
	pendingRole      string
	pendingMessageID string
	pendingTimestamp string
	pendingText      strings.Builder
}

// NewACPTranscriptCollector constructs one ACP transcript collector.
func NewACPTranscriptCollector() *ACPTranscriptCollector {
	return &ACPTranscriptCollector{}
}

// HandleRawUpdate parses and applies one raw ACP session/update payload.
func (c *ACPTranscriptCollector) HandleRawUpdate(raw json.RawMessage) error {
	update, err := ParseACPUpdate(raw)
	if err != nil {
		return err
	}
	c.HandleUpdate(update)
	return nil
}

// HandleUpdate applies one normalized ACP update to the collector.
func (c *ACPTranscriptCollector) HandleUpdate(update ACPUpdate) {
	if c == nil {
		return
	}

	switch update.Type {
	case ACPUpdateTypeUserMessageChunk, ACPUpdateTypeAgentMessageChunk:
		c.handleMessageChunk(update)
	default:
		c.flush()
	}
}

// Result returns the reconstructed transcript.
func (c *ACPTranscriptCollector) Result() SessionTranscriptResult {
	if c == nil {
		return SessionTranscriptResult{}
	}
	c.flush()
	return CloneSessionTranscriptResult(SessionTranscriptResult{
		Messages: c.messages,
	})
}

func (c *ACPTranscriptCollector) handleMessageChunk(update ACPUpdate) {
	role := normalizeTranscriptRole(update)
	if role == "" {
		return
	}
	if update.MessageID != "" && update.MessageID != c.pendingMessageID {
		c.flush()
	}
	if c.pendingRole != "" && c.pendingRole != role {
		c.flush()
	}
	if c.pendingRole == "" {
		c.pendingRole = role
	}
	if c.pendingMessageID == "" {
		c.pendingMessageID = strings.TrimSpace(update.MessageID)
	}
	if c.pendingTimestamp == "" {
		c.pendingTimestamp = strings.TrimSpace(update.Timestamp)
	}
	c.pendingText.WriteString(update.Delta)
}

func (c *ACPTranscriptCollector) flush() {
	if c == nil {
		return
	}

	role := strings.TrimSpace(c.pendingRole)
	content := normalizeTranscriptReplayText(c.pendingText.String())
	if role == "user" || role == "assistant" {
		if content != "" {
			c.messages = append(c.messages, SessionTranscriptMessage{
				Role:      role,
				Content:   content,
				Timestamp: strings.TrimSpace(c.pendingTimestamp),
			})
		}
	}

	c.pendingRole = ""
	c.pendingMessageID = ""
	c.pendingTimestamp = ""
	c.pendingText.Reset()
}

func normalizeTranscriptRole(update ACPUpdate) string {
	switch strings.TrimSpace(strings.ToLower(update.Role)) {
	case "user", "assistant":
		return strings.TrimSpace(strings.ToLower(update.Role))
	}

	switch update.Type {
	case ACPUpdateTypeUserMessageChunk:
		return "user"
	case ACPUpdateTypeAgentMessageChunk:
		return "assistant"
	default:
		return ""
	}
}

func normalizeTranscriptReplayText(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return strings.TrimSpace(content)
}
