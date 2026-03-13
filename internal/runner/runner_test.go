// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestEscalationForPolicy(t *testing.T) {
	tests := []struct {
		policy string
		want   DivergenceAction
	}{
		{"pause", DivergencePause},
		{"intensify", DivergenceIntensify},
		{"migrate", DivergenceMigrate},
		{"log", DivergenceLog},
		{"", DivergenceLog},
		{"unknown", DivergenceLog},
	}
	for _, tt := range tests {
		got := EscalationForPolicy(tt.policy)
		if got != tt.want {
			t.Errorf("EscalationForPolicy(%q) = %d, want %d", tt.policy, got, tt.want)
		}
	}
}

func TestLoadManifestData_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	mPath := filepath.Join(dir, "test.manifest.json")
	if err := os.WriteFile(mPath, []byte(`{"capabilities":{"clock":{"version":1}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	data := LoadManifestData("irrelevant.wasm", mPath, testLogger())
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

	data := LoadManifestData(wasmPath, "", testLogger())
	if string(data) != `{"capabilities":{}}` {
		t.Fatalf("unexpected manifest: %s", data)
	}
}

func TestLoadManifestData_NoManifestFallsBack(t *testing.T) {
	data := LoadManifestData("/tmp/nonexistent.wasm", "", testLogger())
	if string(data) != "{}" {
		t.Fatalf("expected empty JSON, got: %s", data)
	}
}

func TestLoadManifestData_NonWASMPath(t *testing.T) {
	data := LoadManifestData("agent", "", testLogger())
	if string(data) != "{}" {
		t.Fatalf("expected empty JSON for non-.wasm path, got: %s", data)
	}
}
