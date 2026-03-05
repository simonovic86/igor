// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewEngine(t *testing.T) {
	ctx := context.Background()
	engine, err := NewEngine(ctx, newTestLogger())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close(ctx)
}

func TestRuntime_Getter(t *testing.T) {
	ctx := context.Background()
	engine, err := NewEngine(ctx, newTestLogger())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close(ctx)

	rt := engine.Runtime()
	if rt == nil {
		t.Fatal("Runtime() returned nil")
	}
}

func TestLoadWASM_InvalidPath(t *testing.T) {
	ctx := context.Background()
	engine, err := NewEngine(ctx, newTestLogger())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close(ctx)

	_, err = engine.LoadWASM(ctx, "/nonexistent/path.wasm")
	if err == nil {
		t.Fatal("expected error for nonexistent WASM path")
	}
}

func TestLoadWASM_InvalidBinary(t *testing.T) {
	ctx := context.Background()
	engine, err := NewEngine(ctx, newTestLogger())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close(ctx)

	// Write garbage data to a temp file
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.wasm")
	if err := os.WriteFile(path, []byte("not a wasm module"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err = engine.LoadWASM(ctx, path)
	if err == nil {
		t.Fatal("expected error for invalid WASM binary")
	}
}

func TestLoadWASM_MinimalModule(t *testing.T) {
	ctx := context.Background()
	engine, err := NewEngine(ctx, newTestLogger())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close(ctx)

	// Minimal valid WASM module: magic + version (empty module)
	minimalWASM := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}

	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.wasm")
	if err := os.WriteFile(path, minimalWASM, 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	compiled, err := engine.LoadWASM(ctx, path)
	if err != nil {
		t.Fatalf("LoadWASM: %v", err)
	}
	if compiled == nil {
		t.Fatal("LoadWASM returned nil compiled module")
	}
}

func TestInstantiateModule(t *testing.T) {
	ctx := context.Background()
	engine, err := NewEngine(ctx, newTestLogger())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close(ctx)

	// Minimal valid WASM module
	minimalWASM := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.wasm")
	if err := os.WriteFile(path, minimalWASM, 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	compiled, err := engine.LoadWASM(ctx, path)
	if err != nil {
		t.Fatalf("LoadWASM: %v", err)
	}

	mod, err := engine.InstantiateModule(ctx, compiled, "test-module")
	if err != nil {
		t.Fatalf("InstantiateModule: %v", err)
	}
	defer mod.Close(ctx)

	if mod.Name() != "test-module" {
		t.Errorf("module name: got %q, want %q", mod.Name(), "test-module")
	}
}

func TestEngine_CloseIdempotent(t *testing.T) {
	ctx := context.Background()
	engine, err := NewEngine(ctx, newTestLogger())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	// Close should work the first time
	if err := engine.Close(ctx); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	// Second close should not panic (wazero returns nil on double-close)
	if err := engine.Close(ctx); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
