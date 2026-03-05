// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/simonovic86/igor/internal/agent"
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

func TestHandleDivergenceAction_None(t *testing.T) {
	ctx := context.Background()
	stop := HandleDivergenceAction(ctx, nil, nil, DivergenceNone, nil, testLogger())
	if stop {
		t.Error("DivergenceNone should not stop the loop")
	}
}

func TestHandleDivergenceAction_Log(t *testing.T) {
	ctx := context.Background()
	stop := HandleDivergenceAction(ctx, nil, nil, DivergenceLog, nil, testLogger())
	if stop {
		t.Error("DivergenceLog should not stop the loop")
	}
}

func TestHandleDivergenceAction_MigrateWithNilFn(t *testing.T) {
	ctx := context.Background()
	// With nil migrateFn, DivergenceMigrate should still stop the loop (pause fallback).
	// We use a minimal instance to avoid nil dereference in SaveCheckpointToStorage.
	inst := &agent.Instance{AgentID: "test-agent"}
	stop := HandleDivergenceAction(ctx, inst, nil, DivergenceMigrate, nil, testLogger())
	if !stop {
		t.Error("DivergenceMigrate with nil migrateFn should stop the loop")
	}
}

func TestHandleDivergenceAction_MigrateWithFn_Success(t *testing.T) {
	ctx := context.Background()
	inst := &agent.Instance{AgentID: "test-agent"}
	migrateFn := func(_ context.Context, _ string) error { return nil }
	stop := HandleDivergenceAction(ctx, inst, nil, DivergenceMigrate, migrateFn, testLogger())
	if !stop {
		t.Error("successful migration should stop the loop")
	}
}

func TestHandleDivergenceAction_MigrateWithFn_Failure(t *testing.T) {
	ctx := context.Background()
	inst := &agent.Instance{AgentID: "test-agent"}
	migrateFn := func(_ context.Context, _ string) error { return fmt.Errorf("no peers") }
	stop := HandleDivergenceAction(ctx, inst, nil, DivergenceMigrate, migrateFn, testLogger())
	if !stop {
		t.Error("failed migration should still stop the loop (pause fallback)")
	}
}
