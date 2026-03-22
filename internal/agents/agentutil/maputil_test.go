package agentutil_test

import (
	"testing"

	"github.com/beyond5959/ngent/internal/agents/agentutil"
)

func TestMapString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values map[string]any
		key    string
		want   string
	}{
		{name: "nil map", values: nil, key: "k", want: ""},
		{name: "missing key", values: map[string]any{"a": "b"}, key: "k", want: ""},
		{name: "present string", values: map[string]any{"k": "hello"}, key: "k", want: "hello"},
		{name: "non-string value", values: map[string]any{"k": 42}, key: "k", want: ""},
		{name: "bool value", values: map[string]any{"k": true}, key: "k", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := agentutil.MapString(tc.values, tc.key); got != tc.want {
				t.Fatalf("MapString() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMapStringSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values map[string]any
		key    string
		want   []string
	}{
		{name: "nil map", values: nil, key: "k", want: nil},
		{name: "missing key", values: map[string]any{"a": "b"}, key: "k", want: nil},
		{name: "typed []string", values: map[string]any{"k": []string{"a", "b"}}, key: "k", want: []string{"a", "b"}},
		{name: "json []any of strings", values: map[string]any{"k": []any{"x", "y", "z"}}, key: "k", want: []string{"x", "y", "z"}},
		{name: "[]any mixed types skips non-string", values: map[string]any{"k": []any{"x", 42, "z"}}, key: "k", want: []string{"x", "z"}},
		{name: "wrong type int", values: map[string]any{"k": 99}, key: "k", want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := agentutil.MapStringSlice(tc.values, tc.key)
			if len(got) != len(tc.want) {
				t.Fatalf("MapStringSlice() len = %d, want %d (got=%v want=%v)", len(got), len(tc.want), got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("MapStringSlice()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestMapInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values map[string]any
		key    string
		want   int
	}{
		{name: "nil map", values: nil, key: "k", want: 0},
		{name: "missing key", values: map[string]any{"a": 1}, key: "k", want: 0},
		{name: "int value", values: map[string]any{"k": int(7)}, key: "k", want: 7},
		{name: "int64 value", values: map[string]any{"k": int64(100)}, key: "k", want: 100},
		{name: "float64 json number", values: map[string]any{"k": float64(42)}, key: "k", want: 42},
		{name: "string not int", values: map[string]any{"k": "nope"}, key: "k", want: 0},
		{name: "bool not int", values: map[string]any{"k": true}, key: "k", want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := agentutil.MapInt(tc.values, tc.key); got != tc.want {
				t.Fatalf("MapInt() = %d, want %d", got, tc.want)
			}
		})
	}
}
