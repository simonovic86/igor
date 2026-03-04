package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestLoadManifestData_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	mPath := filepath.Join(dir, "test.manifest.json")
	if err := os.WriteFile(mPath, []byte(`{"capabilities":{"clock":{"version":1}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	data := loadManifestData("irrelevant.wasm", mPath, testLogger())
	if string(data) != `{"capabilities":{"clock":{"version":1}}}` {
		t.Fatalf("unexpected manifest: %s", data)
	}
}

func TestLoadManifestData_DerivedFromWASMPath(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "agent.wasm")
	mPath := filepath.Join(dir, "agent.manifest.json")
	if err := os.WriteFile(mPath, []byte(`{"capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	data := loadManifestData(wasmPath, "", testLogger())
	if string(data) != `{"capabilities":{}}` {
		t.Fatalf("unexpected manifest: %s", data)
	}
}

func TestLoadManifestData_NoManifestFallsBack(t *testing.T) {
	data := loadManifestData("/tmp/nonexistent.wasm", "", testLogger())
	if string(data) != "{}" {
		t.Fatalf("expected empty JSON, got: %s", data)
	}
}

func TestLoadManifestData_NonWASMPath(t *testing.T) {
	// When wasmPath doesn't end in .wasm and no explicit manifest, should fallback
	data := loadManifestData("agent", "", testLogger())
	if string(data) != "{}" {
		t.Fatalf("expected empty JSON for non-.wasm path, got: %s", data)
	}
}

func TestEscalationForPolicy(t *testing.T) {
	tests := []struct {
		policy string
		want   divergenceAction
	}{
		{"pause", divergencePause},
		{"intensify", divergenceIntensify},
		{"migrate", divergenceMigrate},
		{"log", divergenceLog},
		{"", divergenceLog},
		{"unknown", divergenceLog},
	}
	for _, tt := range tests {
		got := escalationForPolicy(tt.policy)
		if got != tt.want {
			t.Errorf("escalationForPolicy(%q) = %d, want %d", tt.policy, got, tt.want)
		}
	}
}
