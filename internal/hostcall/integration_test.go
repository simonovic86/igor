package hostcall

import (
	"context"
	"encoding/binary"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/simonovic86/igor/pkg/manifest"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// testAgentSource is a minimal TinyGo agent that imports igor hostcalls.
// Compiled with TinyGo to produce a WASI reactor module (_initialize, no proc_exit).
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

// buildTestAgent compiles the test agent source to WASM using TinyGo.
// Returns the path to the compiled WASM binary, or skips the test.
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

// setupTestModule builds, instantiates, and initializes a test agent with full capabilities.
func setupTestModule(t *testing.T) (wazero.Runtime, api.Module, *eventlog.EventLog) {
	t.Helper()
	wasmPath := buildTestAgent(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	el := eventlog.NewEventLog(0)
	rt := wazero.NewRuntime(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	m := &manifest.CapabilityManifest{
		Capabilities: map[string]manifest.CapabilityConfig{
			"clock": {Version: 1},
			"rand":  {Version: 1},
			"log":   {Version: 1},
		},
	}

	reg := NewRegistry(logger, el)
	if err := reg.RegisterHostModule(ctx, rt, m); err != nil {
		t.Fatalf("RegisterHostModule: %v", err)
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

	return rt, mod, el
}

func verifyTickLog(t *testing.T, sealed *eventlog.TickLog, expectedEntries int) {
	t.Helper()
	if sealed == nil {
		t.Fatal("expected sealed tick log")
	}
	if len(sealed.Entries) != expectedEntries {
		t.Fatalf("expected %d event log entries, got %d", expectedEntries, len(sealed.Entries))
	}
}

func TestIntegration_HostcallsEndToEnd(t *testing.T) {
	rt, mod, el := setupTestModule(t)
	ctx := context.Background()
	defer rt.Close(ctx)
	defer mod.Close(ctx)

	initFn := mod.ExportedFunction("agent_init")
	if initFn == nil {
		t.Fatal("agent_init not exported")
	}
	if _, err := initFn.Call(ctx); err != nil {
		t.Fatalf("agent_init: %v", err)
	}

	// Tick 1: verify all three observation types recorded
	el.BeginTick(1)
	tickFn := mod.ExportedFunction("agent_tick")
	if tickFn == nil {
		t.Fatal("agent_tick not exported")
	}
	if _, err := tickFn.Call(ctx); err != nil {
		t.Fatalf("agent_tick: %v", err)
	}
	sealed := el.SealTick()
	verifyTickLog(t, sealed, 3)

	// Verify clock_now entry
	if sealed.Entries[0].HostcallID != eventlog.ClockNow {
		t.Errorf("entry 0: expected ClockNow, got %d", sealed.Entries[0].HostcallID)
	}
	clockValue := binary.LittleEndian.Uint64(sealed.Entries[0].Payload)
	if clockValue == 0 {
		t.Error("clock value should be non-zero")
	}

	// Verify rand_bytes entry
	if sealed.Entries[1].HostcallID != eventlog.RandBytes {
		t.Errorf("entry 1: expected RandBytes, got %d", sealed.Entries[1].HostcallID)
	}

	// Verify log_emit entry
	if sealed.Entries[2].HostcallID != eventlog.LogEmit {
		t.Errorf("entry 2: expected LogEmit, got %d", sealed.Entries[2].HostcallID)
	}
	if string(sealed.Entries[2].Payload) != "tick" {
		t.Errorf("log payload: expected 'tick', got %q", string(sealed.Entries[2].Payload))
	}

	// Tick 2: verify history accumulates
	el.BeginTick(2)
	if _, err := tickFn.Call(ctx); err != nil {
		t.Fatalf("agent_tick (2): %v", err)
	}
	sealed2 := el.SealTick()
	verifyTickLog(t, sealed2, 3)

	history := el.History()
	if len(history) != 2 {
		t.Errorf("expected 2 ticks in history, got %d", len(history))
	}
}

func TestIntegration_DenyByDefault(t *testing.T) {
	wasmPath := buildTestAgent(t)

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	el := eventlog.NewEventLog(0)

	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	// Register only "clock" — no rand, no log
	m := &manifest.CapabilityManifest{
		Capabilities: map[string]manifest.CapabilityConfig{
			"clock": {Version: 1},
		},
	}

	reg := NewRegistry(logger, el)
	if err := reg.RegisterHostModule(ctx, rt, m); err != nil {
		t.Fatalf("RegisterHostModule: %v", err)
	}

	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatalf("read WASM: %v", err)
	}

	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		// Some wazero versions fail at compile time for missing imports
		t.Logf("Compile failed with missing imports (expected): %v", err)
		return
	}

	// Instantiation should fail because rand_bytes and log_emit are not in the igor module
	_, err = rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
		WithName("test-agent-deny").
		WithStartFunctions())
	if err == nil {
		t.Error("expected instantiation to fail when agent imports undeclared capabilities")
	} else {
		t.Logf("Correctly denied undeclared capabilities: %v", err)
	}
}
