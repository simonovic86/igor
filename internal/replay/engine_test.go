// SPDX-License-Identifier: Apache-2.0

package replay

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/simonovic86/igor/pkg/manifest"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
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

// testAgentNoHostcalls is a minimal agent with no igor hostcall imports.
const testAgentNoHostcalls = `package main

import "unsafe"

var counter uint64

//export agent_init
func agent_init() { counter = 0 }

//export agent_tick
func agent_tick() uint32 { counter++; return 0 }

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

func buildAgent(t *testing.T, source string) string {
	t.Helper()

	tinygoTool, err := exec.LookPath("tinygo")
	if err != nil {
		t.Skip("tinygo not found, skipping integration test")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.go")
	wasmPath := filepath.Join(dir, "agent.wasm")

	if err := os.WriteFile(srcPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write agent source: %v", err)
	}

	goMod := "module testagent\n\ngo 1.24\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	cmd := exec.Command(tinygoTool, "build", "-target=wasi", "-no-debug", "-o", wasmPath, ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("failed to build test WASM agent: %s\n%s", err, out)
	}

	return wasmPath
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// liveTickEnv sets up a live wazero environment to execute ticks and capture event logs.
type liveTickEnv struct {
	rt       wazero.Runtime
	mod      api.Module
	el       *eventlog.EventLog
	wasmPath string
}

func newLiveTickEnv(t *testing.T, source string, capManifest *manifest.CapabilityManifest) *liveTickEnv {
	t.Helper()
	wasmPath := buildAgent(t, source)
	ctx := context.Background()
	logger := newTestLogger()

	el := eventlog.NewEventLog(0)
	rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().
		WithMemoryLimitPages(1024).
		WithCloseOnContextDone(true))

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		t.Fatalf("instantiate WASI: %v", err)
	}

	// Register live hostcalls (only if manifest has capabilities)
	if capManifest != nil && len(capManifest.Capabilities) > 0 {
		if err := registerLiveHostModule(ctx, rt, capManifest, el, logger); err != nil {
			t.Fatalf("registerLiveHostModule: %v", err)
		}
	}

	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatalf("read WASM: %v", err)
	}

	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		t.Fatalf("CompileModule: %v", err)
	}

	mod, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
		WithName("test-agent").
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStartFunctions())
	if err != nil {
		t.Fatalf("InstantiateModule: %v", err)
	}

	if initFn := mod.ExportedFunction("_initialize"); initFn != nil {
		if _, err := initFn.Call(ctx); err != nil {
			t.Fatalf("_initialize: %v", err)
		}
	}

	return &liveTickEnv{rt: rt, mod: mod, el: el, wasmPath: wasmPath}
}

func (env *liveTickEnv) close(ctx context.Context) {
	env.mod.Close(ctx)
	env.rt.Close(ctx)
}

func (env *liveTickEnv) init(t *testing.T, ctx context.Context) {
	t.Helper()
	fn := env.mod.ExportedFunction("agent_init")
	if _, err := fn.Call(ctx); err != nil {
		t.Fatalf("agent_init: %v", err)
	}
}

func (env *liveTickEnv) tick(t *testing.T, ctx context.Context, tickNum uint64) *eventlog.TickLog {
	t.Helper()
	env.el.BeginTick(tickNum)
	fn := env.mod.ExportedFunction("agent_tick")
	if _, err := fn.Call(ctx); err != nil {
		t.Fatalf("agent_tick (tick %d): %v", tickNum, err)
	}
	sealed := env.el.SealTick()
	if sealed == nil {
		t.Fatalf("SealTick returned nil for tick %d", tickNum)
	}
	return sealed
}

func (env *liveTickEnv) checkpoint(t *testing.T, ctx context.Context) []byte {
	t.Helper()
	fnSize := env.mod.ExportedFunction("agent_checkpoint")
	sizeResults, err := fnSize.Call(ctx)
	if err != nil {
		t.Fatalf("agent_checkpoint: %v", err)
	}
	size := uint32(sizeResults[0])
	if size == 0 {
		return []byte{}
	}

	fnPtr := env.mod.ExportedFunction("agent_checkpoint_ptr")
	ptrResults, err := fnPtr.Call(ctx)
	if err != nil {
		t.Fatalf("agent_checkpoint_ptr: %v", err)
	}
	ptr := uint32(ptrResults[0])

	data, ok := env.mod.Memory().Read(ptr, size)
	if !ok {
		t.Fatal("failed to read checkpoint from memory")
	}

	out := make([]byte, len(data))
	copy(out, data)
	return out
}

func (env *liveTickEnv) wasmBytes(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(env.wasmPath)
	if err != nil {
		t.Fatalf("read WASM bytes: %v", err)
	}
	return b
}

// registerLiveHostModule registers live (non-replay) hostcalls for the test environment.
// This mirrors internal/hostcall.Registry but is self-contained to avoid import cycles.
func registerLiveHostModule(
	ctx context.Context,
	rt wazero.Runtime,
	m *manifest.CapabilityManifest,
	el *eventlog.EventLog,
	logger *slog.Logger,
) error {
	builder := rt.NewHostModuleBuilder("igor")
	registered := 0

	if m.Has("clock") {
		builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context) int64 {
				// Use a fixed timestamp for deterministic live execution in tests
				now := int64(1000000000)
				payload := binary.LittleEndian.AppendUint64(nil, uint64(now))
				el.Record(eventlog.ClockNow, payload)
				return now
			}).
			Export("clock_now")
		registered++
	}

	if m.Has("rand") {
		builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, mod api.Module, ptr, length uint32) int32 {
				// Use deterministic bytes for test replay
				buf := make([]byte, length)
				for i := range buf {
					buf[i] = byte(i + 1)
				}
				el.Record(eventlog.RandBytes, buf)
				if !mod.Memory().Write(ptr, buf) {
					return -4
				}
				return 0
			}).
			Export("rand_bytes")
		registered++
	}

	if m.Has("log") {
		builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, mod api.Module, ptr, length uint32) {
				data, ok := mod.Memory().Read(ptr, length)
				if !ok {
					return
				}
				p := make([]byte, len(data))
				copy(p, data)
				el.Record(eventlog.LogEmit, p)
				logger.Info("[agent]", "msg", string(data))
			}).
			Export("log_emit")
		registered++
	}

	if registered == 0 {
		return nil
	}

	_, err := builder.Instantiate(ctx)
	return err
}

func fullManifest() *manifest.CapabilityManifest {
	return &manifest.CapabilityManifest{
		Capabilities: map[string]manifest.CapabilityConfig{
			"clock": {Version: 1},
			"rand":  {Version: 1},
			"log":   {Version: 1},
		},
	}
}

func emptyManifest() *manifest.CapabilityManifest {
	return &manifest.CapabilityManifest{
		Capabilities: map[string]manifest.CapabilityConfig{},
	}
}

func TestReplayTick_Verified(t *testing.T) {
	ctx := context.Background()
	env := newLiveTickEnv(t, testAgentSource, fullManifest())
	defer env.close(ctx)

	env.init(t, ctx)

	// Checkpoint at tick 0
	stateN := env.checkpoint(t, ctx)

	// Execute tick 1 and capture event log
	tickLog := env.tick(t, ctx, 1)

	// Checkpoint at tick 1
	stateN1 := env.checkpoint(t, ctx)

	// Verify counter advanced from 0 to 1
	counter := binary.LittleEndian.Uint64(stateN1)
	if counter != 1 {
		t.Fatalf("expected counter=1, got %d", counter)
	}

	// Replay tick 1
	engine := NewEngine(newTestLogger())
	result := engine.ReplayTick(ctx, env.wasmBytes(t), fullManifest(), stateN, tickLog, stateN1)

	if result.Error != nil {
		t.Fatalf("replay error: %v", result.Error)
	}
	if !result.Verified {
		t.Fatalf("replay not verified, first diff at byte %d", result.FirstDiffByte)
	}
}

func TestReplayTick_Divergence(t *testing.T) {
	ctx := context.Background()
	env := newLiveTickEnv(t, testAgentSource, fullManifest())
	defer env.close(ctx)

	env.init(t, ctx)
	stateN := env.checkpoint(t, ctx)
	tickLog := env.tick(t, ctx, 1)
	stateN1 := env.checkpoint(t, ctx)

	// Tamper with expected state
	tampered := make([]byte, len(stateN1))
	copy(tampered, stateN1)
	tampered[0] ^= 0xFF

	engine := NewEngine(newTestLogger())
	result := engine.ReplayTick(ctx, env.wasmBytes(t), fullManifest(), stateN, tickLog, tampered)

	if result.Error != nil {
		t.Fatalf("replay error: %v", result.Error)
	}
	if result.Verified {
		t.Fatal("replay should not be verified with tampered state")
	}
	if result.FirstDiffByte < 0 {
		t.Fatal("FirstDiffByte should be >= 0 for divergence")
	}
}

func TestReplayTick_MultipleTicks(t *testing.T) {
	ctx := context.Background()
	env := newLiveTickEnv(t, testAgentSource, fullManifest())
	defer env.close(ctx)

	env.init(t, ctx)

	// Capture checkpoints and tick logs for 3 ticks
	states := make([][]byte, 4)
	tickLogs := make([]*eventlog.TickLog, 3)

	states[0] = env.checkpoint(t, ctx)
	for i := 0; i < 3; i++ {
		tickLogs[i] = env.tick(t, ctx, uint64(i+1))
		states[i+1] = env.checkpoint(t, ctx)
	}

	// Replay each tick independently
	engine := NewEngine(newTestLogger())
	wasmBytes := env.wasmBytes(t)

	for i := 0; i < 3; i++ {
		result := engine.ReplayTick(ctx, wasmBytes, fullManifest(), states[i], tickLogs[i], states[i+1])
		if result.Error != nil {
			t.Fatalf("tick %d replay error: %v", i+1, result.Error)
		}
		if !result.Verified {
			t.Fatalf("tick %d not verified, first diff at byte %d", i+1, result.FirstDiffByte)
		}
	}

	// Verify final counter is 3
	finalCounter := binary.LittleEndian.Uint64(states[3])
	if finalCounter != 3 {
		t.Fatalf("expected final counter=3, got %d", finalCounter)
	}
}

func TestReplayTick_HostcallMismatch(t *testing.T) {
	ctx := context.Background()
	env := newLiveTickEnv(t, testAgentSource, fullManifest())
	defer env.close(ctx)

	env.init(t, ctx)
	stateN := env.checkpoint(t, ctx)
	tickLog := env.tick(t, ctx, 1)
	stateN1 := env.checkpoint(t, ctx)

	// Corrupt the first entry's HostcallID (ClockNow → RandBytes)
	corruptedEntries := make([]eventlog.Entry, len(tickLog.Entries))
	copy(corruptedEntries, tickLog.Entries)
	corruptedEntries[0].HostcallID = eventlog.RandBytes

	corruptedLog := &eventlog.TickLog{
		TickNumber: tickLog.TickNumber,
		Entries:    corruptedEntries,
	}

	engine := NewEngine(newTestLogger())
	result := engine.ReplayTick(ctx, env.wasmBytes(t), fullManifest(), stateN, corruptedLog, stateN1)

	if result.Error == nil {
		t.Fatal("expected error for hostcall mismatch")
	}
	t.Logf("Got expected error: %v", result.Error)
}

func TestReplayTick_EmptyEventLog(t *testing.T) {
	ctx := context.Background()
	env := newLiveTickEnv(t, testAgentNoHostcalls, emptyManifest())
	defer env.close(ctx)

	// Initialize
	fn := env.mod.ExportedFunction("agent_init")
	if _, err := fn.Call(ctx); err != nil {
		t.Fatalf("agent_init: %v", err)
	}

	stateN := env.checkpoint(t, ctx)

	// Tick without event log (no hostcalls to record)
	tickFn := env.mod.ExportedFunction("agent_tick")
	if _, err := tickFn.Call(ctx); err != nil {
		t.Fatalf("agent_tick: %v", err)
	}

	stateN1 := env.checkpoint(t, ctx)

	// Create empty tick log
	emptyTickLog := &eventlog.TickLog{
		TickNumber: 1,
		Entries:    nil,
	}

	engine := NewEngine(newTestLogger())
	result := engine.ReplayTick(ctx, env.wasmBytes(t), emptyManifest(), stateN, emptyTickLog, stateN1)

	if result.Error != nil {
		t.Fatalf("replay error: %v", result.Error)
	}
	if !result.Verified {
		t.Fatalf("replay not verified, first diff at byte %d", result.FirstDiffByte)
	}

	// Verify counter advanced
	counter := binary.LittleEndian.Uint64(stateN1)
	if counter != 1 {
		t.Fatalf("expected counter=1, got %d", counter)
	}
}

func TestReplayTick_ExhaustedLog(t *testing.T) {
	ctx := context.Background()
	env := newLiveTickEnv(t, testAgentSource, fullManifest())
	defer env.close(ctx)

	env.init(t, ctx)
	stateN := env.checkpoint(t, ctx)
	tickLog := env.tick(t, ctx, 1)
	stateN1 := env.checkpoint(t, ctx)

	// Remove the last entry (agent will call log_emit but iterator is exhausted)
	truncatedEntries := make([]eventlog.Entry, len(tickLog.Entries)-1)
	copy(truncatedEntries, tickLog.Entries[:len(tickLog.Entries)-1])

	truncatedLog := &eventlog.TickLog{
		TickNumber: tickLog.TickNumber,
		Entries:    truncatedEntries,
	}

	engine := NewEngine(newTestLogger())
	result := engine.ReplayTick(ctx, env.wasmBytes(t), fullManifest(), stateN, truncatedLog, stateN1)

	if result.Error == nil {
		t.Fatal("expected error for exhausted event log")
	}
	t.Logf("Got expected error: %v", result.Error)
}

func TestReplayChain_ThreeTicks(t *testing.T) {
	ctx := context.Background()
	env := newLiveTickEnv(t, testAgentSource, fullManifest())
	defer env.close(ctx)

	env.init(t, ctx)

	initialState := env.checkpoint(t, ctx)
	chainSnaps := make([]ChainSnapshot, 3)
	for i := 0; i < 3; i++ {
		tickLog := env.tick(t, ctx, uint64(i+1))
		chainSnaps[i] = ChainSnapshot{
			TickNumber: uint64(i + 1),
			TickLog:    tickLog,
		}
	}
	finalState := env.checkpoint(t, ctx)
	finalHash := sha256.Sum256(finalState)

	engine := NewEngine(newTestLogger())
	result := engine.ReplayChain(ctx, env.wasmBytes(t), fullManifest(), initialState, chainSnaps, finalHash)

	if result.Error != nil {
		t.Fatalf("chain replay error: %v", result.Error)
	}
	if !result.Verified {
		t.Fatal("chain replay not verified")
	}
	if result.TicksReplayed != 3 {
		t.Errorf("TicksReplayed: got %d, want 3", result.TicksReplayed)
	}
}

func TestReplayChain_SingleTick(t *testing.T) {
	ctx := context.Background()
	env := newLiveTickEnv(t, testAgentSource, fullManifest())
	defer env.close(ctx)

	env.init(t, ctx)

	initialState := env.checkpoint(t, ctx)
	tickLog := env.tick(t, ctx, 1)
	finalState := env.checkpoint(t, ctx)
	finalHash := sha256.Sum256(finalState)

	engine := NewEngine(newTestLogger())
	result := engine.ReplayChain(ctx, env.wasmBytes(t), fullManifest(), initialState, []ChainSnapshot{
		{TickNumber: 1, TickLog: tickLog},
	}, finalHash)

	if result.Error != nil {
		t.Fatalf("chain replay error: %v", result.Error)
	}
	if !result.Verified {
		t.Fatal("chain replay not verified")
	}
	if result.TicksReplayed != 1 {
		t.Errorf("TicksReplayed: got %d, want 1", result.TicksReplayed)
	}
}

func TestReplayChain_Divergence(t *testing.T) {
	ctx := context.Background()
	env := newLiveTickEnv(t, testAgentSource, fullManifest())
	defer env.close(ctx)

	env.init(t, ctx)

	initialState := env.checkpoint(t, ctx)
	chainSnaps := make([]ChainSnapshot, 3)
	for i := 0; i < 3; i++ {
		tickLog := env.tick(t, ctx, uint64(i+1))
		chainSnaps[i] = ChainSnapshot{
			TickNumber: uint64(i + 1),
			TickLog:    tickLog,
		}
	}

	wrongHash := [32]byte{0xFF}

	engine := NewEngine(newTestLogger())
	result := engine.ReplayChain(ctx, env.wasmBytes(t), fullManifest(), initialState, chainSnaps, wrongHash)

	if result.Error != nil {
		t.Fatalf("chain replay error: %v", result.Error)
	}
	if result.Verified {
		t.Fatal("chain replay should not be verified with wrong hash")
	}
}

func TestReplayChain_Empty(t *testing.T) {
	engine := NewEngine(newTestLogger())
	result := engine.ReplayChain(context.Background(), nil, nil, nil, nil, [32]byte{})
	if result.Error == nil {
		t.Fatal("expected error for empty snapshots")
	}
}

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

func TestReplayTick_Timeout(t *testing.T) {
	wasmPath := buildAgent(t, testAgentInfiniteLoop)
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatalf("read WASM: %v", err)
	}

	initialState := make([]byte, 8)
	emptyTickLog := &eventlog.TickLog{TickNumber: 1, Entries: nil}

	engine := NewEngine(newTestLogger())
	defer engine.Close(context.Background())

	start := time.Now()
	result := engine.ReplayTick(context.Background(), wasmBytes, emptyManifest(), initialState, emptyTickLog, nil)
	elapsed := time.Since(start)

	if result.Error == nil {
		t.Fatal("expected error from timeout, got nil")
	}
	t.Logf("Replay tick timed out after %v with error: %v", elapsed, result.Error)

	// Should complete within a reasonable bound (allow generous margin for CI)
	if elapsed > replayTickTimeout+5*time.Second {
		t.Fatalf("replay took too long: %v (expected < %v)", elapsed, replayTickTimeout+5*time.Second)
	}
}

func TestReplayTick_UnconsumedEntries(t *testing.T) {
	ctx := context.Background()
	env := newLiveTickEnv(t, testAgentSource, fullManifest())
	defer env.close(ctx)

	env.init(t, ctx)
	stateN := env.checkpoint(t, ctx)
	tickLog := env.tick(t, ctx, 1)
	stateN1 := env.checkpoint(t, ctx)

	// Add a spurious entry
	extraEntries := make([]eventlog.Entry, len(tickLog.Entries)+1)
	copy(extraEntries, tickLog.Entries)
	extraEntries[len(tickLog.Entries)] = eventlog.Entry{
		HostcallID: eventlog.ClockNow,
		Payload:    make([]byte, 8),
	}

	paddedLog := &eventlog.TickLog{
		TickNumber: tickLog.TickNumber,
		Entries:    extraEntries,
	}

	engine := NewEngine(newTestLogger())
	result := engine.ReplayTick(ctx, env.wasmBytes(t), fullManifest(), stateN, paddedLog, stateN1)

	if result.Error == nil {
		t.Fatal("expected error for unconsumed entries")
	}
	t.Logf("Got expected error: %v", result.Error)
}
