package gemini

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadUserAuthTypePrefersSettingsOverEnv(t *testing.T) {
	dir := t.TempDir()
	writeSettings(t, dir, `{"security":{"auth":{"selectedType":"oauth-personal"}}}`)
	t.Setenv("GEMINI_API_KEY", "invalid-token")

	got := readUserAuthType(dir)
	if got != "oauth-personal" {
		t.Fatalf("readUserAuthType()=%q, want %q", got, "oauth-personal")
	}
}

func TestReadUserAuthTypeUsesLegacySelectedAuthType(t *testing.T) {
	dir := t.TempDir()
	writeSettings(t, dir, `{"selectedAuthType":"gemini-api-key"}`)
	t.Setenv("GEMINI_API_KEY", "")

	got := readUserAuthType(dir)
	if got != "gemini-api-key" {
		t.Fatalf("readUserAuthType()=%q, want %q", got, "gemini-api-key")
	}
}

func TestReadUserAuthTypeFallsBackToEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GEMINI_API_KEY", "from-env")

	got := readUserAuthType(dir)
	if got != "gemini-api-key" {
		t.Fatalf("readUserAuthType()=%q, want %q", got, "gemini-api-key")
	}
}

func TestReadUserAuthTypeDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GEMINI_API_KEY", "")

	got := readUserAuthType(dir)
	if got != "oauth-personal" {
		t.Fatalf("readUserAuthType()=%q, want %q", got, "oauth-personal")
	}
}

func writeSettings(t *testing.T, geminiDir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(geminiDir, "settings.json"), []byte(content), 0o600); err != nil {
		t.Fatalf("write settings: %v", err)
	}
}
