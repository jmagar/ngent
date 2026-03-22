package agents

import (
	"context"
	"testing"
)

// ---- PromptContentBlock / PromptResource / TurnPromptConfig types ----

func TestPromptContentBlockFields(t *testing.T) {
	t.Parallel()

	b := PromptContentBlock{
		Type:     "text",
		Text:     "hello",
		Data:     "base64data",
		URI:      "file:///foo",
		Path:     "/foo",
		Name:     "foo.txt",
		MimeType: "text/plain",
		Range:    &ByteRange{Start: 0, End: 10},
	}
	if b.Type != "text" {
		t.Fatalf("Type = %q, want %q", b.Type, "text")
	}
	if b.Range == nil || b.Range.Start != 0 || b.Range.End != 10 {
		t.Fatalf("Range = %v, want {0,10}", b.Range)
	}
}

func TestPromptResourceFields(t *testing.T) {
	t.Parallel()

	r := PromptResource{
		Name:     "schema.sql",
		URI:      "file:///schema.sql",
		Path:     "/schema.sql",
		MimeType: "text/plain",
		Text:     "CREATE TABLE ...",
		Data:     "",
		Range:    &ByteRange{Start: 5, End: 20},
	}
	if r.Name != "schema.sql" {
		t.Fatalf("Name = %q, want schema.sql", r.Name)
	}
	if r.Range == nil || r.Range.Start != 5 || r.Range.End != 20 {
		t.Fatalf("Range = %v, want {5,20}", r.Range)
	}
}

func TestTurnPromptConfigFields(t *testing.T) {
	t.Parallel()

	cfg := TurnPromptConfig{
		Profile:            "fast",
		ApprovalPolicy:     "auto",
		Sandbox:            "none",
		Personality:        "assistant",
		SystemInstructions: "Be concise.",
	}
	if cfg.Profile != "fast" {
		t.Fatalf("Profile = %q, want fast", cfg.Profile)
	}
	if cfg.SystemInstructions != "Be concise." {
		t.Fatalf("SystemInstructions = %q, want \"Be concise.\"", cfg.SystemInstructions)
	}
}

// ---- TurnContent context helpers ----

func TestWithTurnContentRoundTrip(t *testing.T) {
	t.Parallel()

	blocks := []PromptContentBlock{
		{Type: "text", Text: "hello"},
		{Type: "image", URI: "file:///img.png"},
	}
	ctx := WithTurnContent(context.Background(), blocks)
	got := TurnContentFromContext(ctx)
	if got == nil {
		t.Fatal("TurnContentFromContext() = nil, want content")
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].Text != "hello" {
		t.Fatalf("got[0].Text = %q, want hello", got[0].Text)
	}
}

func TestWithTurnContentEmptySliceIsNoop(t *testing.T) {
	t.Parallel()

	ctx := WithTurnContent(context.Background(), []PromptContentBlock{})
	got := TurnContentFromContext(ctx)
	if got != nil {
		t.Fatalf("TurnContentFromContext(empty) = %v, want nil", got)
	}
}

func TestTurnContentFromContextNil(t *testing.T) {
	t.Parallel()

	//nolint:staticcheck
	got := TurnContentFromContext(nil)
	if got != nil {
		t.Fatalf("TurnContentFromContext(nil) = %v, want nil", got)
	}
}

// ---- TurnResources context helpers ----

func TestWithTurnResourcesRoundTrip(t *testing.T) {
	t.Parallel()

	resources := []PromptResource{
		{Name: "file.go", Path: "/tmp/file.go"},
	}
	ctx := WithTurnResources(context.Background(), resources)
	got := TurnResourcesFromContext(ctx)
	if got == nil {
		t.Fatal("TurnResourcesFromContext() = nil, want resources")
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Name != "file.go" {
		t.Fatalf("got[0].Name = %q, want file.go", got[0].Name)
	}
}

func TestWithTurnResourcesEmptySliceIsNoop(t *testing.T) {
	t.Parallel()

	ctx := WithTurnResources(context.Background(), []PromptResource{})
	got := TurnResourcesFromContext(ctx)
	if got != nil {
		t.Fatalf("TurnResourcesFromContext(empty) = %v, want nil", got)
	}
}

func TestTurnResourcesFromContextNil(t *testing.T) {
	t.Parallel()

	//nolint:staticcheck
	got := TurnResourcesFromContext(nil)
	if got != nil {
		t.Fatalf("TurnResourcesFromContext(nil) = %v, want nil", got)
	}
}

// ---- TurnPromptConfig context helpers ----

func TestWithTurnPromptConfigRoundTrip(t *testing.T) {
	t.Parallel()

	cfg := TurnPromptConfig{
		Profile:        "fast",
		ApprovalPolicy: "auto",
	}
	ctx := WithTurnPromptConfig(context.Background(), cfg)
	got, ok := TurnPromptConfigFromContext(ctx)
	if !ok {
		t.Fatal("TurnPromptConfigFromContext() ok = false, want true")
	}
	if got.Profile != "fast" {
		t.Fatalf("got.Profile = %q, want fast", got.Profile)
	}
	if got.ApprovalPolicy != "auto" {
		t.Fatalf("got.ApprovalPolicy = %q, want auto", got.ApprovalPolicy)
	}
}

func TestTurnPromptConfigFromContextMissing(t *testing.T) {
	t.Parallel()

	_, ok := TurnPromptConfigFromContext(context.Background())
	if ok {
		t.Fatal("TurnPromptConfigFromContext() ok = true for empty context, want false")
	}
}

func TestTurnPromptConfigFromContextNil(t *testing.T) {
	t.Parallel()

	//nolint:staticcheck
	_, ok := TurnPromptConfigFromContext(nil)
	if ok {
		t.Fatal("TurnPromptConfigFromContext(nil) ok = true, want false")
	}
}

// ---- AdapterInfo type ----

func TestAdapterInfoFields(t *testing.T) {
	t.Parallel()

	a := AdapterInfo{Name: "codex-acp", Version: "0.3.3"}
	if a.Name != "codex-acp" {
		t.Fatalf("Name = %q, want codex-acp", a.Name)
	}
	if a.Version != "0.3.3" {
		t.Fatalf("Version = %q, want 0.3.3", a.Version)
	}
}
