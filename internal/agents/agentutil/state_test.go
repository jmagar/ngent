package agentutil_test

import (
	"testing"

	"github.com/beyond5959/ngent/internal/agents"
	"github.com/beyond5959/ngent/internal/agents/agentutil"
)

func TestNewState(t *testing.T) {
	state, err := agentutil.NewState("kimi", agentutil.Config{
		Dir:     "  /tmp/workspace  ",
		ModelID: " kimi-k2 ",
		ConfigOverrides: map[string]string{
			"reasoning": " high ",
			"model":     "ignored",
			"empty":     " ",
		},
	})
	if err != nil {
		t.Fatalf("NewState() unexpected error: %v", err)
	}
	if got, want := state.Dir(), "/tmp/workspace"; got != want {
		t.Fatalf("Dir() = %q, want %q", got, want)
	}
	if got, want := state.CurrentModelID(), "kimi-k2"; got != want {
		t.Fatalf("CurrentModelID() = %q, want %q", got, want)
	}
	overrides := state.CurrentConfigOverrides()
	if got, want := len(overrides), 1; got != want {
		t.Fatalf("len(CurrentConfigOverrides()) = %d, want %d", got, want)
	}
	if got, want := overrides["reasoning"], "high"; got != want {
		t.Fatalf("reasoning = %q, want %q", got, want)
	}
}

func TestStateApplyConfigOptionResult(t *testing.T) {
	state, err := agentutil.NewState("opencode", agentutil.Config{Dir: "/tmp"})
	if err != nil {
		t.Fatalf("NewState() unexpected error: %v", err)
	}

	state.ApplyConfigOptionResult("model", "gpt-5", []agents.ConfigOption{{
		ID:           "model",
		CurrentValue: "gpt-5-mini",
	}})
	if got, want := state.CurrentModelID(), "gpt-5-mini"; got != want {
		t.Fatalf("CurrentModelID() = %q, want %q", got, want)
	}

	state.ApplyConfigOptionResult("reasoning", "medium", []agents.ConfigOption{{
		ID:           "reasoning",
		CurrentValue: "high",
	}})
	if got, want := state.CurrentConfigOverrides()["reasoning"], "high"; got != want {
		t.Fatalf("reasoning = %q, want %q", got, want)
	}
}
