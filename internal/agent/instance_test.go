// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/simonovic86/igor/internal/eventlog"
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
func agent_tick() uint32 {
	counter++
	_ = clockNow()
	var buf [4]byte
	randBytes(uint32(uintptr(unsafe.Pointer(&buf[0]))), 4)
	msg := []byte("tick")
	logEmit(uint32(uintptr(unsafe.Pointer(&msg[0]))), uint32(len(msg)))
	return 0
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

// testAgentInfiniteLoop is an agent whose agent_tick never returns.
const testAgentInfiniteLoop = `package main

import "unsafe"

var counter uint64

//export agent_init
func agent_init() { counter = 0 }

//export agent_tick
func agent_tick() uint32 { for { counter++ } }

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
	return buildTestAgentFromSource(t, testAgentSource)
}

func buildTestAgentFromSource(t *testing.T, source string) string {
	t.Helper()

	tinygoTool, err := exec.LookPath("tinygo")
	if err != nil {
		t.Skip("tinygo not found, skipping integration test")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.go")
	wasmPath := filepath.Join(dir, "agent.wasm")

	if err := os.WriteFile(srcPath, []byte(source), 0o644); err != nil {
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

	instance, err := LoadAgent(ctx, engine, wasmPath, "test-agent", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), manifest, nil, "", nil, logger)
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
	_, err = LoadAgent(ctx, engine, wasmPath, "test-empty", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), []byte("{}"), nil, "", nil, logger)
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

	_, err = LoadAgent(ctx, engine, wasmPath, "test-bad-manifest", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), []byte("not json"), nil, "", nil, logger)
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
	instance, err := LoadAgent(ctx, engine, wasmPath, "test-tick", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), manifest, nil, "", nil, logger)
	if err != nil {
		t.Fatalf("LoadAgent: %v", err)
	}
	defer instance.Close(ctx)

	if err := instance.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if _, err := instance.Tick(ctx); err != nil {
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
	if snap.PostStateHash == ([32]byte{}) {
		t.Error("snapshot PostStateHash should not be zero")
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
	instance, err := LoadAgent(ctx, engine, wasmPath, "test-budget", storageProvider, 0, budget.FromFloat(0.01), manifest, nil, "", nil, logger)
	if err != nil {
		t.Fatalf("LoadAgent: %v", err)
	}
	defer instance.Close(ctx)

	if err := instance.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	_, err = instance.Tick(ctx)
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
	instance, err := LoadAgent(ctx, engine, wasmPath, "test-ckpt", storageProvider, budget.FromFloat(10.0), priceMicrocents, manifest, nil, "", nil, logger)
	if err != nil {
		t.Fatalf("LoadAgent: %v", err)
	}
	defer instance.Close(ctx)

	if err := instance.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Run a few ticks
	for i := 0; i < 3; i++ {
		if _, err := instance.Tick(ctx); err != nil {
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

	// Verify checkpoint format v0x03: [version:1][budget:8][price:8][tick:8][wasmHash:32][majorVer:8][leaseGen:8][leaseExpiry:8][state:N]
	if len(rawCheckpoint) < 81 {
		t.Fatalf("checkpoint too short: %d bytes", len(rawCheckpoint))
	}
	if rawCheckpoint[0] != 0x03 {
		t.Fatalf("checkpoint version: got %d, want 3", rawCheckpoint[0])
	}

	storedBudget := int64(binary.LittleEndian.Uint64(rawCheckpoint[1:9]))
	storedPrice := int64(binary.LittleEndian.Uint64(rawCheckpoint[9:17]))
	storedTick := binary.LittleEndian.Uint64(rawCheckpoint[17:25])

	if storedBudget != savedBudget {
		t.Errorf("stored budget: got %d, want %d", storedBudget, savedBudget)
	}
	if storedPrice != priceMicrocents {
		t.Errorf("stored price: got %d, want %d", storedPrice, priceMicrocents)
	}
	if storedTick != 3 {
		t.Errorf("stored tick number: got %d, want 3", storedTick)
	}

	// Verify WASM hash is present (bytes 25-57)
	var storedHash [32]byte
	copy(storedHash[:], rawCheckpoint[25:57])
	if storedHash == [32]byte{} {
		t.Error("stored WASM hash should not be zero")
	}

	// State portion should be 8 bytes (uint64 counter) — starts at offset 81 for v0x03
	state := rawCheckpoint[81:]
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
	instance, err := LoadAgent(ctx, engine, wasmPath, "test-evict", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), manifest, nil, "", nil, logger)
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
		if _, err := instance.Tick(ctx); err != nil {
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

func TestReplayWindow_WeightedEviction(t *testing.T) {
	// Manually construct an instance with a replay window to test
	// that low-observation snapshots are evicted before high-observation ones.
	inst := &Instance{
		replayWindowMax: 3,
		logger:          newTestLogger(),
	}

	// Simulate a window of 3 snapshots with varying observation counts.
	inst.ReplayWindow = []TickSnapshot{
		{TickNumber: 1, TickLog: &eventlog.TickLog{Entries: make([]eventlog.Entry, 5)}}, // high
		{TickNumber: 2, TickLog: &eventlog.TickLog{Entries: nil}},                       // zero
		{TickNumber: 3, TickLog: &eventlog.TickLog{Entries: make([]eventlog.Entry, 3)}}, // medium
	}

	// Append a 4th snapshot, triggering eviction. The snapshot with tick 2
	// (zero observations) should be evicted, not the oldest (tick 1).
	inst.ReplayWindow = append(inst.ReplayWindow, TickSnapshot{
		TickNumber:    4,
		TickLog:       &eventlog.TickLog{Entries: make([]eventlog.Entry, 2)},
		PostStateHash: [32]byte{},
	})

	// Run eviction logic (same as in Tick)
	maxSnaps := inst.replayWindowMax
	if len(inst.ReplayWindow) > maxSnaps {
		evictIdx := 0
		evictScore := inst.ReplayWindow[0].observationScore()
		for j := 1; j < len(inst.ReplayWindow)-1; j++ {
			score := inst.ReplayWindow[j].observationScore()
			if score < evictScore {
				evictScore = score
				evictIdx = j
			}
		}
		inst.ReplayWindow = append(inst.ReplayWindow[:evictIdx], inst.ReplayWindow[evictIdx+1:]...)
	}

	if len(inst.ReplayWindow) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(inst.ReplayWindow))
	}

	// Tick 2 (zero observations) should have been evicted.
	// Remaining: ticks 1, 3, 4.
	expectedTicks := []uint64{1, 3, 4}
	for i, expected := range expectedTicks {
		if inst.ReplayWindow[i].TickNumber != expected {
			t.Errorf("ReplayWindow[%d].TickNumber: got %d, want %d", i, inst.ReplayWindow[i].TickNumber, expected)
		}
	}
}

func TestTickSnapshot_ObservationScore(t *testing.T) {
	tests := []struct {
		name  string
		snap  TickSnapshot
		score int
	}{
		{"nil_ticklog", TickSnapshot{TickLog: nil}, 0},
		{"empty_entries", TickSnapshot{TickLog: &eventlog.TickLog{Entries: nil}}, 0},
		{"three_entries", TickSnapshot{TickLog: &eventlog.TickLog{Entries: make([]eventlog.Entry, 3)}}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.snap.observationScore(); got != tt.score {
				t.Errorf("observationScore(): got %d, want %d", got, tt.score)
			}
		})
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

func TestParseCheckpointHeader_Golden(t *testing.T) {
	data, err := os.ReadFile("testdata/checkpoint.bin")
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}

	hdr, state, err := ParseCheckpointHeader(data)
	if err != nil {
		t.Fatalf("ParseCheckpointHeader: %v", err)
	}

	if hdr.Budget != 1000000 {
		t.Errorf("budget: got %d, want 1000000", hdr.Budget)
	}
	if hdr.PricePerSecond != 1000 {
		t.Errorf("price: got %d, want 1000", hdr.PricePerSecond)
	}
	if hdr.TickNumber != 5 {
		t.Errorf("tick: got %d, want 5", hdr.TickNumber)
	}

	expectedHash := sha256.Sum256([]byte("known-wasm-binary-for-golden-test"))
	if hdr.WASMHash != expectedHash {
		t.Errorf("wasmHash mismatch")
	}

	// v0x02 fixture should return zero epoch
	if hdr.Epoch.MajorVersion != 0 || hdr.Epoch.LeaseGeneration != 0 {
		t.Errorf("v0x02 epoch: got %s, want (0,0)", hdr.Epoch)
	}

	if len(state) != 8 {
		t.Fatalf("state length: got %d, want 8", len(state))
	}
	counter := binary.LittleEndian.Uint64(state)
	if counter != 3 {
		t.Errorf("counter: got %d, want 3", counter)
	}
}

func TestParseCheckpointHeader_V3Golden(t *testing.T) {
	data, err := os.ReadFile("testdata/checkpoint_v3.bin")
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}

	hdr, state, err := ParseCheckpointHeader(data)
	if err != nil {
		t.Fatalf("ParseCheckpointHeader: %v", err)
	}

	if hdr.Budget != 2000000 {
		t.Errorf("budget: got %d, want 2000000", hdr.Budget)
	}
	if hdr.PricePerSecond != 1500 {
		t.Errorf("price: got %d, want 1500", hdr.PricePerSecond)
	}
	if hdr.TickNumber != 10 {
		t.Errorf("tick: got %d, want 10", hdr.TickNumber)
	}

	expectedHash := sha256.Sum256([]byte("known-wasm-binary-for-golden-test"))
	if hdr.WASMHash != expectedHash {
		t.Errorf("wasmHash mismatch")
	}

	if hdr.Epoch.MajorVersion != 3 {
		t.Errorf("majorVersion: got %d, want 3", hdr.Epoch.MajorVersion)
	}
	if hdr.Epoch.LeaseGeneration != 7 {
		t.Errorf("leaseGeneration: got %d, want 7", hdr.Epoch.LeaseGeneration)
	}
	if hdr.LeaseExpiry != 1700000000000000000 {
		t.Errorf("leaseExpiry: got %d, want 1700000000000000000", hdr.LeaseExpiry)
	}

	if len(state) != 8 {
		t.Fatalf("state length: got %d, want 8", len(state))
	}
	counter := binary.LittleEndian.Uint64(state)
	if counter != 5 {
		t.Errorf("counter: got %d, want 5", counter)
	}
}

func TestParseCheckpointHeader_EmptyState(t *testing.T) {
	data, err := os.ReadFile("testdata/checkpoint_empty_state.bin")
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}

	hdr, state, err := ParseCheckpointHeader(data)
	if err != nil {
		t.Fatalf("ParseCheckpointHeader: %v", err)
	}

	if hdr.Budget != 500000 {
		t.Errorf("budget: got %d, want 500000", hdr.Budget)
	}
	if hdr.PricePerSecond != 2000 {
		t.Errorf("price: got %d, want 2000", hdr.PricePerSecond)
	}
	if hdr.TickNumber != 0 {
		t.Errorf("tick: got %d, want 0", hdr.TickNumber)
	}
	if len(state) != 0 {
		t.Errorf("state should be empty, got %d bytes", len(state))
	}
}

func TestParseCheckpointHeader_NegativeBudget(t *testing.T) {
	checkpoint := make([]byte, 57)
	checkpoint[0] = 0x02
	// Write -1 as budget (all bits set via two's complement)
	negBudget := int64(-1)
	binary.LittleEndian.PutUint64(checkpoint[1:9], uint64(negBudget))
	binary.LittleEndian.PutUint64(checkpoint[9:17], 1000)

	_, _, err := ParseCheckpointHeader(checkpoint)
	if err == nil {
		t.Error("expected error for negative budget in checkpoint")
	}
}

func TestParseCheckpointHeader_NegativePrice(t *testing.T) {
	checkpoint := make([]byte, 57)
	checkpoint[0] = 0x02
	binary.LittleEndian.PutUint64(checkpoint[1:9], 1000000)
	negPrice := int64(-500)
	binary.LittleEndian.PutUint64(checkpoint[9:17], uint64(negPrice))

	_, _, err := ParseCheckpointHeader(checkpoint)
	if err == nil {
		t.Error("expected error for negative price in checkpoint")
	}
}

func TestParseCheckpointHeader_Corruption(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"zero_length", []byte{}},
		{"too_short", make([]byte, 30)},
		{"truncated_hash", make([]byte, 40)},
		{"wrong_version", func() []byte {
			b := make([]byte, 57)
			b[0] = 0xFF
			return b
		}()},
		{"one_byte", []byte{0x02}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseCheckpointHeader(tt.input)
			if err == nil {
				t.Error("expected error for corrupted checkpoint")
			}
		})
	}
}

// testAgentAlternate is a different agent (adds 2 instead of 1) to produce a
// different WASM binary hash for hash-mismatch testing.
const testAgentAlternate = `package main

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
func agent_tick() uint32 {
	counter += 2
	_ = clockNow()
	var buf [4]byte
	randBytes(uint32(uintptr(unsafe.Pointer(&buf[0]))), 4)
	msg := []byte("tick")
	logEmit(uint32(uintptr(unsafe.Pointer(&msg[0]))), uint32(len(msg)))
	return 0
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

func TestLoadCheckpointFromStorage_WASMHashMismatch(t *testing.T) {
	// Build two different WASM agents to get different hashes.
	wasmPath1 := buildTestAgent(t)
	wasmPath2 := buildTestAgentFromSource(t, testAgentAlternate)

	ctx := context.Background()
	logger := newTestLogger()

	engine, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close(ctx)

	storageDir := t.TempDir()
	storageProvider, err := storage.NewFSProvider(storageDir, logger)
	if err != nil {
		t.Fatalf("NewFSProvider: %v", err)
	}

	manifest := []byte(`{"capabilities":{"clock":{"version":1},"rand":{"version":1},"log":{"version":1}}}`)

	// Load agent 1, run a tick, save checkpoint.
	inst1, err := LoadAgent(ctx, engine, wasmPath1, "hash-test", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), manifest, nil, "", nil, logger)
	if err != nil {
		t.Fatalf("LoadAgent(1): %v", err)
	}
	if err := inst1.Init(ctx); err != nil {
		t.Fatalf("Init(1): %v", err)
	}
	if _, err := inst1.Tick(ctx); err != nil {
		t.Fatalf("Tick(1): %v", err)
	}
	if err := inst1.SaveCheckpointToStorage(ctx); err != nil {
		t.Fatalf("SaveCheckpoint(1): %v", err)
	}
	inst1.Close(ctx)

	// Load agent 2 with the same agent ID — different binary, different hash.
	inst2, err := LoadAgent(ctx, engine, wasmPath2, "hash-test", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), manifest, nil, "", nil, logger)
	if err != nil {
		t.Fatalf("LoadAgent(2): %v", err)
	}
	defer inst2.Close(ctx)

	if err := inst2.Init(ctx); err != nil {
		t.Fatalf("Init(2): %v", err)
	}

	// This should fail: checkpoint was created by agent 1, but we're loading with agent 2.
	err = inst2.LoadCheckpointFromStorage(ctx)
	if err == nil {
		t.Fatal("expected WASM hash mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "WASM hash mismatch") {
		t.Fatalf("expected hash mismatch error, got: %v", err)
	}
}

func TestLoadAgent_ExcessiveMemoryRejected(t *testing.T) {
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

	// Manifest declaring 128MB — exceeds the node's 64MB limit.
	manifest := []byte(`{
		"capabilities": {"clock":{"version":1},"rand":{"version":1},"log":{"version":1}},
		"resource_limits": {"max_memory_bytes": 134217728}
	}`)

	_, err = LoadAgent(ctx, engine, wasmPath, "test-mem", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), manifest, nil, "", nil, logger)
	if err == nil {
		t.Error("expected error when agent requires more memory than node provides")
	}
}

func TestLoadAgent_ValidMemoryAccepted(t *testing.T) {
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

	// Manifest declaring 32MB — within the node's 64MB limit.
	manifest := []byte(`{
		"capabilities": {"clock":{"version":1},"rand":{"version":1},"log":{"version":1}},
		"resource_limits": {"max_memory_bytes": 33554432}
	}`)

	instance, err := LoadAgent(ctx, engine, wasmPath, "test-mem-ok", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), manifest, nil, "", nil, logger)
	if err != nil {
		t.Fatalf("LoadAgent: %v", err)
	}
	defer instance.Close(ctx)

	if instance.FullManifest == nil {
		t.Error("FullManifest should not be nil")
	}
	if instance.FullManifest.ResourceLimits.MaxMemoryBytes != 33554432 {
		t.Errorf("MaxMemoryBytes: got %d, want 33554432", instance.FullManifest.ResourceLimits.MaxMemoryBytes)
	}
}

func TestTick_TimeoutEnforcement(t *testing.T) {
	// Build an agent with an infinite loop in agent_tick.
	// The 100ms context timeout (enforced via wazero's WithCloseOnContextDone)
	// should interrupt execution and return an error.
	wasmPath := buildTestAgentFromSource(t, testAgentInfiniteLoop)
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

	// No hostcall imports needed — the infinite-loop agent doesn't use them.
	instance, err := LoadAgent(ctx, engine, wasmPath, "test-timeout", storageProvider, budget.FromFloat(10.0), budget.FromFloat(0.01), nil, nil, "", nil, logger)
	if err != nil {
		t.Fatalf("LoadAgent: %v", err)
	}
	defer instance.Close(ctx)

	if err := instance.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	start := time.Now()
	_, err = instance.Tick(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from timeout, got nil")
	}
	t.Logf("Tick timed out after %v with error: %v", elapsed, err)

	// Should complete within a reasonable bound (timeout is 100ms, allow up to 1s for CI)
	if elapsed > 1*time.Second {
		t.Fatalf("tick took too long: %v (expected < 1s)", elapsed)
	}
}
