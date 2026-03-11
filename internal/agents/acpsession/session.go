package acpsession

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/beyond5959/ngent/internal/agents"
)

// Capabilities describes ACP session support discovered during initialize.
type Capabilities struct {
	CanList bool
	CanLoad bool
}

// ParseInitializeCapabilities extracts ACP session capabilities from initialize.
func ParseInitializeCapabilities(raw json.RawMessage) Capabilities {
	var payload struct {
		AgentCapabilities map[string]json.RawMessage `json:"agentCapabilities"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &payload) != nil {
		return Capabilities{}
	}

	caps := Capabilities{}
	if enabledFeature(payload.AgentCapabilities["loadSession"]) {
		caps.CanLoad = true
	}

	var sessionCaps map[string]json.RawMessage
	if rawSessionCaps, ok := payload.AgentCapabilities["sessionCapabilities"]; ok {
		_ = json.Unmarshal(rawSessionCaps, &sessionCaps)
	}
	if enabledFeature(sessionCaps["list"]) {
		caps.CanList = true
	}
	if enabledFeature(sessionCaps["load"]) || enabledFeature(sessionCaps["resume"]) {
		caps.CanLoad = true
	}
	return caps
}

// ParseSessionListResult decodes one ACP session/list result payload.
func ParseSessionListResult(raw json.RawMessage) (agents.SessionListResult, error) {
	var payload struct {
		Sessions []struct {
			SessionID string         `json:"sessionId"`
			CWD       string         `json:"cwd"`
			Title     string         `json:"title"`
			UpdatedAt string         `json:"updatedAt"`
			Meta      map[string]any `json:"_meta"`
		} `json:"sessions"`
		NextCursor string `json:"nextCursor"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return agents.SessionListResult{}, fmt.Errorf("decode session/list result: %w", err)
	}

	result := agents.SessionListResult{
		NextCursor: strings.TrimSpace(payload.NextCursor),
		Sessions:   make([]agents.SessionInfo, 0, len(payload.Sessions)),
	}
	for _, session := range payload.Sessions {
		result.Sessions = append(result.Sessions, agents.SessionInfo{
			SessionID: session.SessionID,
			CWD:       session.CWD,
			Title:     session.Title,
			UpdatedAt: session.UpdatedAt,
			Meta:      session.Meta,
		})
	}
	return agents.CloneSessionListResult(result), nil
}

func enabledFeature(raw json.RawMessage) bool {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return false
	}
	switch string(raw) {
	case "null", "false", `""`:
		return false
	}

	var boolValue bool
	if err := json.Unmarshal(raw, &boolValue); err == nil {
		return boolValue
	}
	return true
}
