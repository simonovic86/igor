package agent

import (
	"context"
	"encoding/binary"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/simonovic86/igor/internal/runtime"
	"github.com/simonovic86/igor/internal/storage"
	"github.com/simonovic86/igor/pkg/budget"
)

// testAgentSource is a minimal TinyGo agent that imports igor hostcalls.
const testAgentSource = `package main

import "unsafe"

//go:wasmimport igor clock_now
func clockNow() int64

//go:wasmimport igor rand_bytes
func randBytes(ptr uint32, length uint32) int32

//go:wasmimport igor log_emit
func logEmit(ptr uint32, length uint32)

var counter uint64

//export agent_init
func agent_init() { counter = 0 }

//export agent_tick
func agent_tick() {
	counter++
	_ = clockNow()
	var buf [4]byte
	randBytes(uint32(uintptr(unsafe.Pointer(&buf[0]))), 4)
	msg := []byte("tick")
	logEmit(uint32(uintptr(unsafe.Pointer(&msg[0]))), uint32(len(msg)))
}

//export agent_checkpoint
func agent_checkpoint() uint32 { return 8 }

//export agent_checkpoint_ptr
func agent_checkpoint_ptr() uint32 {
	return uint32(uintptr(unsafe.Pointer(&counter)))
}

//export agent_resume
func agent_resume(ptr, size uint32) {
	if size >= 8 {
		buf := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
		counter = *(*uint64)(unsafe.Pointer(&buf[0]))
	}
}

func main() {}
`

func buildTestAgent(t *testing.T) string {
	t.Helper()

	tinygoTool, err := exec.LookPath("tinygo")
	if err != nil {
		t.Skip("tinygo not found, skipping integration test")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.go")
	wasmPath := filepath.Join(dir, "agent.wasm")

	if err := os.WriteFile(srcPath, []byte(testAgentSource), 0o644); err != nil {
		t.Fatalf("write test agent source: %v", err)
	}

	goMod := "module testagent\n\ngo 1.24\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	cmd := exec.Command(tinygoTool, "build", "-target=wasi", "-no-debug", "-o", wasmPath, ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("failed to build test WASM agent with tinygo: %s\n%s", err, out)
	}

	return wasmPath
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestLoadAgent_WithManifest(t *testing.T) {
	wasmPath := buildTestAgent(t)
	ctx := context.Background()
	logger := newTestLogger()

	engine, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close(ctx)

	storageProvider, err := storage.NewFSProvider(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("NewFSProvider: %v", err)
	}

	manifest := []byte(`{"capabilities":{"clock":{"version":1},"rand":{"version":1},"log":{"version":1}}}`)

	instance, err := LoadAgent(ctx, engine, wasmPath, "test-agent", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), manifest, logger)
	if err != nil {
		t.Fatalf("LoadAgent: %v", err)
	}
	defer instance.Close(ctx)

	if instance.AgentID != "test-agent" {
		t.Errorf("AgentID: got %q, want %q", instance.AgentID, "test-agent")
	}
	if instance.Budget != budget.FromFloat(10.0) {
		t.Errorf("Budget: got %d, want %d", instance.Budget, budget.FromFloat(10.0))
	}
	if instance.Manifest == nil {
		t.Error("Manifest should not be nil")
	}
	if instance.EventLog == nil {
		t.Error("EventLog should not be nil")
	}
}

func TestLoadAgent_EmptyManifest(t *testing.T) {
	wasmPath := buildTestAgent(t)
	ctx := context.Background()
	logger := newTestLogger()

	engine, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close(ctx)

	storageProvider, err := storage.NewFSProvider(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("NewFSProvider: %v", err)
	}

	// Empty manifest — backward compatible, but agent imports hostcalls
	// so instantiation should fail (deny by default)
	_, err = LoadAgent(ctx, engine, wasmPath, "test-empty", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), []byte("{}"), logger)
	if err == nil {
		t.Error("expected error when agent imports undeclared capabilities")
	}
}

func TestLoadAgent_InvalidManifest(t *testing.T) {
	wasmPath := buildTestAgent(t)
	ctx := context.Background()
	logger := newTestLogger()

	engine, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close(ctx)

	storageProvider, err := storage.NewFSProvider(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("NewFSProvider: %v", err)
	}

	_, err = LoadAgent(ctx, engine, wasmPath, "test-bad-manifest", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), []byte("not json"), logger)
	if err == nil {
		t.Error("expected error for invalid JSON manifest")
	}
}

func TestTick_RecordObservations(t *testing.T) {
	wasmPath := buildTestAgent(t)
	ctx := context.Background()
	logger := newTestLogger()

	engine, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close(ctx)

	storageProvider, err := storage.NewFSProvider(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("NewFSProvider: %v", err)
	}

	manifest := []byte(`{"capabilities":{"clock":{"version":1},"rand":{"version":1},"log":{"version":1}}}`)
	instance, err := LoadAgent(ctx, engine, wasmPath, "test-tick", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), manifest, logger)
	if err != nil {
		t.Fatalf("LoadAgent: %v", err)
	}
	defer instance.Close(ctx)

	if err := instance.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if err := instance.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}

	// Verify tick counter advanced
	if instance.TickNumber != 1 {
		t.Errorf("TickNumber: got %d, want 1", instance.TickNumber)
	}

	// Verify event log has entries (clock, rand, log = 3 entries)
	history := instance.EventLog.History()
	if len(history) != 1 {
		t.Fatalf("expected 1 tick in history, got %d", len(history))
	}
	if len(history[0].Entries) != 3 {
		t.Errorf("expected 3 entries in tick log, got %d", len(history[0].Entries))
	}

	// Verify replay window has one snapshot
	if len(instance.ReplayWindow) != 1 {
		t.Fatalf("expected 1 snapshot in ReplayWindow, got %d", len(instance.ReplayWindow))
	}
	snap := instance.ReplayWindow[0]
	if snap.TickNumber != 1 {
		t.Errorf("snapshot TickNumber: got %d, want 1", snap.TickNumber)
	}
	if snap.PreState == nil {
		t.Error("snapshot PreState should not be nil")
	}
	if snap.PostState == nil {
		t.Error("snapshot PostState should not be nil")
	}
	if snap.TickLog == nil {
		t.Error("snapshot TickLog should not be nil")
	}
}

func TestTick_BudgetExhausted(t *testing.T) {
	wasmPath := buildTestAgent(t)
	ctx := context.Background()
	logger := newTestLogger()

	engine, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close(ctx)

	storageProvider, err := storage.NewFSProvider(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("NewFSProvider: %v", err)
	}

	manifest := []byte(`{"capabilities":{"clock":{"version":1},"rand":{"version":1},"log":{"version":1}}}`)
	instance, err := LoadAgent(ctx, engine, wasmPath, "test-budget", storageProvider, 0, budget.FromFloat(0.01), manifest, logger)
	if err != nil {
		t.Fatalf("LoadAgent: %v", err)
	}
	defer instance.Close(ctx)

	if err := instance.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	err = instance.Tick(ctx)
	if err == nil {
		t.Error("expected error when budget is exhausted")
	}
}

func TestCheckpointAndResume(t *testing.T) {
	wasmPath := buildTestAgent(t)
	ctx := context.Background()
	logger := newTestLogger()

	engine, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close(ctx)

	storageProvider, err := storage.NewFSProvider(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("NewFSProvider: %v", err)
	}

	manifest := []byte(`{"capabilities":{"clock":{"version":1},"rand":{"version":1},"log":{"version":1}}}`)
	priceMicrocents := budget.FromFloat(0.5)
	instance, err := LoadAgent(ctx, engine, wasmPath, "test-ckpt", storageProvider, budget.FromFloat(10.0), priceMicrocents, manifest, logger)
	if err != nil {
		t.Fatalf("LoadAgent: %v", err)
	}
	defer instance.Close(ctx)

	if err := instance.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Run a few ticks
	for i := 0; i < 3; i++ {
		if err := instance.Tick(ctx); err != nil {
			t.Fatalf("Tick %d: %v", i+1, err)
		}
	}

	// Checkpoint to storage (includes budget metadata)
	if err := instance.SaveCheckpointToStorage(ctx); err != nil {
		t.Fatalf("SaveCheckpointToStorage: %v", err)
	}

	savedBudget := instance.Budget

	// Load checkpoint from storage to verify round-trip
	rawCheckpoint, err := storageProvider.LoadCheckpoint(ctx, "test-ckpt")
	if err != nil {
		t.Fatalf("LoadCheckpoint: %v", err)
	}

	// Verify checkpoint v1 format: [version:1][budget:8][pricePerSecond:8][state:...]
	if len(rawCheckpoint) < 17 {
		t.Fatalf("checkpoint too short: %d bytes", len(rawCheckpoint))
	}
	if rawCheckpoint[0] != 0x01 {
		t.Fatalf("checkpoint version: got %d, want 1", rawCheckpoint[0])
	}

	storedBudget := int64(binary.LittleEndian.Uint64(rawCheckpoint[1:9]))
	storedPrice := int64(binary.LittleEndian.Uint64(rawCheckpoint[9:17]))

	if storedBudget != savedBudget {
		t.Errorf("stored budget: got %d, want %d", storedBudget, savedBudget)
	}
	if storedPrice != priceMicrocents {
		t.Errorf("stored price: got %d, want %d", storedPrice, priceMicrocents)
	}

	// State portion should be 8 bytes (uint64 counter)
	state := rawCheckpoint[17:]
	if len(state) != 8 {
		t.Errorf("state size: got %d, want 8", len(state))
	}

	counter := binary.LittleEndian.Uint64(state)
	if counter != 3 {
		t.Errorf("counter in checkpoint: got %d, want 3", counter)
	}
}

func TestReplayWindow_Eviction(t *testing.T) {
	wasmPath := buildTestAgent(t)
	ctx := context.Background()
	logger := newTestLogger()

	engine, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close(ctx)

	storageProvider, err := storage.NewFSProvider(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("NewFSProvider: %v", err)
	}

	manifest := []byte(`{"capabilities":{"clock":{"version":1},"rand":{"version":1},"log":{"version":1}}}`)
	instance, err := LoadAgent(ctx, engine, wasmPath, "test-evict", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), manifest, logger)
	if err != nil {
		t.Fatalf("LoadAgent: %v", err)
	}
	defer instance.Close(ctx)

	instance.SetReplayWindowSize(3)

	if err := instance.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Run 5 ticks — window should retain only the last 3
	for i := 0; i < 5; i++ {
		if err := instance.Tick(ctx); err != nil {
			t.Fatalf("Tick %d: %v", i+1, err)
		}
	}

	if len(instance.ReplayWindow) != 3 {
		t.Fatalf("expected 3 snapshots in ReplayWindow, got %d", len(instance.ReplayWindow))
	}

	// Verify the window contains ticks 3, 4, 5 (oldest evicted)
	expectedTicks := []uint64{3, 4, 5}
	for i, expected := range expectedTicks {
		if instance.ReplayWindow[i].TickNumber != expected {
			t.Errorf("ReplayWindow[%d].TickNumber: got %d, want %d", i, instance.ReplayWindow[i].TickNumber, expected)
		}
	}
}

func TestLatestSnapshot(t *testing.T) {
	inst := &Instance{}

	// Empty window returns nil
	if snap := inst.LatestSnapshot(); snap != nil {
		t.Error("expected nil for empty ReplayWindow")
	}

	// Add snapshots
	inst.ReplayWindow = []TickSnapshot{
		{TickNumber: 1},
		{TickNumber: 2},
		{TickNumber: 3},
	}

	snap := inst.LatestSnapshot()
	if snap == nil {
		t.Fatal("expected non-nil LatestSnapshot")
	}
	if snap.TickNumber != 3 {
		t.Errorf("LatestSnapshot().TickNumber: got %d, want 3", snap.TickNumber)
	}
}
