package agentutil

import "encoding/json"

// IsTruthy reports whether a JSON capability value is truthy.
// ACP encodes session capability switches as either `true` (bool) or an object
// like `{"enabled":true}`, so both forms are accepted.
func IsTruthy(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return b
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return false
	}
	var enabled bool
	if v, ok := obj["enabled"]; ok && json.Unmarshal(v, &enabled) == nil {
		return enabled
	}
	return len(obj) > 0
}
