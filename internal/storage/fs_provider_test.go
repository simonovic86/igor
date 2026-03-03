package storage

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"testing"
)

func newTestProvider(t *testing.T) *FSProvider {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	p, err := NewFSProvider(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("NewFSProvider: %v", err)
	}
	return p
}

func TestSaveAndLoadCheckpoint(t *testing.T) {
	p := newTestProvider(t)
	ctx := context.Background()

	state := []byte("hello checkpoint")
	if err := p.SaveCheckpoint(ctx, "agent-1", state); err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}

	loaded, err := p.LoadCheckpoint(ctx, "agent-1")
	if err != nil {
		t.Fatalf("LoadCheckpoint: %v", err)
	}
	if !bytes.Equal(loaded, state) {
		t.Errorf("loaded state mismatch: got %q, want %q", loaded, state)
	}
}

func TestLoadCheckpoint_NotFound(t *testing.T) {
	p := newTestProvider(t)
	ctx := context.Background()

	_, err := p.LoadCheckpoint(ctx, "nonexistent")
	if err != ErrCheckpointNotFound {
		t.Errorf("expected ErrCheckpointNotFound, got %v", err)
	}
}

func TestDeleteCheckpoint(t *testing.T) {
	p := newTestProvider(t)
	ctx := context.Background()

	// Save, then delete, then verify gone
	if err := p.SaveCheckpoint(ctx, "agent-del", []byte("data")); err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}
	if err := p.DeleteCheckpoint(ctx, "agent-del"); err != nil {
		t.Fatalf("DeleteCheckpoint: %v", err)
	}
	_, err := p.LoadCheckpoint(ctx, "agent-del")
	if err != ErrCheckpointNotFound {
		t.Errorf("expected ErrCheckpointNotFound after delete, got %v", err)
	}
}

func TestDeleteCheckpoint_NotFound(t *testing.T) {
	p := newTestProvider(t)
	ctx := context.Background()

	// Deleting non-existent checkpoint should not error (idempotent)
	if err := p.DeleteCheckpoint(ctx, "nonexistent"); err != nil {
		t.Errorf("DeleteCheckpoint should be idempotent, got %v", err)
	}
}

func TestSaveCheckpoint_Overwrite(t *testing.T) {
	p := newTestProvider(t)
	ctx := context.Background()

	// Save initial state
	if err := p.SaveCheckpoint(ctx, "agent-ow", []byte("v1")); err != nil {
		t.Fatalf("SaveCheckpoint v1: %v", err)
	}

	// Overwrite with new state
	if err := p.SaveCheckpoint(ctx, "agent-ow", []byte("v2")); err != nil {
		t.Fatalf("SaveCheckpoint v2: %v", err)
	}

	loaded, err := p.LoadCheckpoint(ctx, "agent-ow")
	if err != nil {
		t.Fatalf("LoadCheckpoint: %v", err)
	}
	if string(loaded) != "v2" {
		t.Errorf("expected 'v2', got %q", loaded)
	}
}

func TestSaveCheckpoint_EmptyState(t *testing.T) {
	p := newTestProvider(t)
	ctx := context.Background()

	if err := p.SaveCheckpoint(ctx, "agent-empty", []byte{}); err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}

	loaded, err := p.LoadCheckpoint(ctx, "agent-empty")
	if err != nil {
		t.Fatalf("LoadCheckpoint: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected empty state, got %d bytes", len(loaded))
	}
}

func TestCheckpointPath_PathTraversal(t *testing.T) {
	p := newTestProvider(t)
	ctx := context.Background()

	// Attempt path traversal with ../
	err := p.SaveCheckpoint(ctx, "../../etc/passwd", []byte("exploit"))
	if err == nil {
		t.Error("expected error for path traversal, got nil")
	}

	_, err = p.LoadCheckpoint(ctx, "../../../etc/shadow")
	if err == nil {
		t.Error("expected error for path traversal on load, got nil")
	}

	err = p.DeleteCheckpoint(ctx, "../../important")
	if err == nil {
		t.Error("expected error for path traversal on delete, got nil")
	}
}

func TestNewFSProvider_CleansStaleTemp(t *testing.T) {
	dir := t.TempDir()
	// Create stale .tmp files before provider initialization
	stalePath := dir + "/agent-stale.checkpoint.tmp"
	if err := os.WriteFile(stalePath, []byte("stale"), 0o644); err != nil {
		t.Fatalf("create stale file: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	_, err := NewFSProvider(dir, logger)
	if err != nil {
		t.Fatalf("NewFSProvider: %v", err)
	}

	// Stale file should be gone
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Error("stale .tmp file should have been cleaned up")
	}
}

func TestNewFSProvider_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	subDir := dir + "/sub/nested"
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	p, err := NewFSProvider(subDir, logger)
	if err != nil {
		t.Fatalf("NewFSProvider: %v", err)
	}

	// Verify the directory was created and is usable
	ctx := context.Background()
	if err := p.SaveCheckpoint(ctx, "test", []byte("ok")); err != nil {
		t.Fatalf("SaveCheckpoint in nested dir: %v", err)
	}
}
