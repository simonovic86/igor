package simulator_test

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/simonovic86/igor/internal/simulator"
)

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
		t.Fatalf("write agent source: %v", err)
	}

	goMod := "module testagent\n\ngo 1.24\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Write manifest
	manifestJSON := `{"capabilities":{"clock":{"version":1},"rand":{"version":1},"log":{"version":1}}}`
	manifestPath := filepath.Join(dir, "agent.manifest.json")
	if err := os.WriteFile(manifestPath, []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
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

func TestSimulator_BasicRun(t *testing.T) {
	wasmPath := buildTestAgent(t)
	ctx := context.Background()

	result, err := simulator.Run(ctx, simulator.Config{
		WASMPath:      wasmPath,
		Budget:        1.0,
		Ticks:         5,
		Deterministic: true,
		RandSeed:      42,
	}, newTestLogger())
	if err != nil {
		t.Fatalf("simulator.Run: %v", err)
	}

	if result.TicksExecuted != 5 {
		t.Errorf("ticks: got %d, want 5", result.TicksExecuted)
	}
	if result.CheckpointSize != 8 {
		t.Errorf("checkpoint size: got %d, want 8", result.CheckpointSize)
	}
	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
}

func TestSimulator_DeterministicReplay(t *testing.T) {
	wasmPath := buildTestAgent(t)
	ctx := context.Background()

	result, err := simulator.Run(ctx, simulator.Config{
		WASMPath:      wasmPath,
		Budget:        1.0,
		Ticks:         5,
		Deterministic: true,
		RandSeed:      42,
		Verify:        true,
	}, newTestLogger())
	if err != nil {
		t.Fatalf("simulator.Run: %v", err)
	}

	if result.TicksExecuted != 5 {
		t.Errorf("ticks: got %d, want 5", result.TicksExecuted)
	}
	if result.ReplayVerified != 5 {
		t.Errorf("replay verified: got %d, want 5", result.ReplayVerified)
	}
	if result.ReplayFailed != 0 {
		t.Errorf("replay failed: got %d, want 0", result.ReplayFailed)
	}
	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
}

func TestSimulator_BudgetExhaustion(t *testing.T) {
	wasmPath := buildTestAgent(t)
	ctx := context.Background()

	// Very small budget should exhaust quickly.
	result, err := simulator.Run(ctx, simulator.Config{
		WASMPath:       wasmPath,
		Budget:         0.000001, // 1 microcent
		PricePerSecond: 1000.0,   // very expensive
		Ticks:          0,        // run until exhausted
		Deterministic:  true,
	}, newTestLogger())
	if err != nil {
		t.Fatalf("simulator.Run: %v", err)
	}

	// Should have stopped due to budget exhaustion.
	if result.FinalBudget > 0 {
		t.Errorf("expected budget exhausted, got %d remaining", result.FinalBudget)
	}
}

func TestSimulator_Reproducible(t *testing.T) {
	wasmPath := buildTestAgent(t)
	ctx := context.Background()

	cfg := simulator.Config{
		WASMPath:      wasmPath,
		Budget:        1.0,
		Ticks:         3,
		Deterministic: true,
		RandSeed:      99,
	}

	r1, err := simulator.Run(ctx, cfg, newTestLogger())
	if err != nil {
		t.Fatalf("run 1: %v", err)
	}

	r2, err := simulator.Run(ctx, cfg, newTestLogger())
	if err != nil {
		t.Fatalf("run 2: %v", err)
	}

	if r1.TicksExecuted != r2.TicksExecuted {
		t.Errorf("ticks differ: %d vs %d", r1.TicksExecuted, r2.TicksExecuted)
	}
	if r1.CheckpointSize != r2.CheckpointSize {
		t.Errorf("checkpoint size differs: %d vs %d", r1.CheckpointSize, r2.CheckpointSize)
	}
}

func TestSimulator_PrintSummary(t *testing.T) {
	r := &simulator.Result{
		TicksExecuted:  10,
		FinalBudget:    500000,
		BudgetConsumed: 500000,
		CheckpointSize: 28,
		ReplayVerified: 10,
	}
	// Just verify it doesn't panic.
	simulator.PrintSummary(r, newTestLogger())
}
