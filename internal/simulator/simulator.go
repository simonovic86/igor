// Package simulator provides a single-process WASM agent runner for local
// development and testing. It drives the agent tick loop with optional
// deterministic hostcalls and per-tick replay verification.
package simulator

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/simonovic86/igor/internal/hostcall"
	"github.com/simonovic86/igor/internal/replay"
	"github.com/simonovic86/igor/internal/wasmutil"
	"github.com/simonovic86/igor/pkg/budget"
	"github.com/simonovic86/igor/pkg/manifest"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Config holds simulator configuration.
type Config struct {
	WASMPath       string
	ManifestPath   string
	Budget         float64
	PricePerSecond float64
	Ticks          int    // 0 = run until budget exhausted
	Verify         bool   // Per-tick replay verification
	Deterministic  bool   // Fixed clock + seeded rand
	RandSeed       uint64 // Seed when deterministic
	ClockStart     int64  // Starting clock nanos when deterministic
	ClockDelta     int64  // Clock advance per call when deterministic
}

// Result summarizes the simulation outcome.
type Result struct {
	TicksExecuted  int
	FinalBudget    int64
	BudgetConsumed int64
	CheckpointSize int
	ReplayVerified int
	ReplayFailed   int
	Errors         []string
}

// simEnv holds the simulator's runtime environment.
type simEnv struct {
	rt             wazero.Runtime
	mod            api.Module
	el             *eventlog.EventLog
	wasmBytes      []byte
	capManifest    *manifest.CapabilityManifest
	replayEngine   *replay.Engine
	pricePerSecond int64
	logger         *slog.Logger
}

// Run executes the simulator.
func Run(ctx context.Context, cfg Config, logger *slog.Logger) (*Result, error) {
	env, budgetMicrocents, err := setup(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}
	defer env.rt.Close(ctx)
	defer env.mod.Close(ctx)
	if env.replayEngine != nil {
		defer env.replayEngine.Close(ctx)
	}

	initialBudget := budgetMicrocents
	result := &Result{}

	budgetMicrocents = runTickLoop(ctx, env, cfg, result, budgetMicrocents)
	finalize(ctx, env, result)

	result.FinalBudget = budgetMicrocents
	result.BudgetConsumed = initialBudget - budgetMicrocents

	return result, nil
}

// setup creates the WASM runtime, loads the agent, and initializes it.
func setup(ctx context.Context, cfg Config, logger *slog.Logger) (*simEnv, int64, error) {
	wasmBytes, err := os.ReadFile(cfg.WASMPath)
	if err != nil {
		return nil, 0, fmt.Errorf("read WASM: %w", err)
	}

	manifestData := loadManifest(cfg)
	capManifest, err := manifest.ParseCapabilityManifest(manifestData)
	if err != nil {
		return nil, 0, fmt.Errorf("parse manifest: %w", err)
	}

	budgetMicrocents := budget.FromFloat(cfg.Budget)
	pricePerSecond := budget.FromFloat(cfg.PricePerSecond)
	if pricePerSecond == 0 {
		pricePerSecond = 1000
	}

	rtConfig := wazero.NewRuntimeConfig().
		WithMemoryLimitPages(1024).
		WithCloseOnContextDone(true)
	rt := wazero.NewRuntimeWithConfig(ctx, rtConfig)

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	el := eventlog.NewEventLog(eventlog.DefaultMaxTicks)

	if err := registerHostcalls(ctx, cfg, rt, capManifest, el, logger); err != nil {
		rt.Close(ctx)
		return nil, 0, err
	}

	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		rt.Close(ctx)
		return nil, 0, fmt.Errorf("compile WASM: %w", err)
	}

	mod, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
		WithName("simulator").
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStartFunctions())
	if err != nil {
		rt.Close(ctx)
		return nil, 0, fmt.Errorf("instantiate module: %w", err)
	}

	if initFn := mod.ExportedFunction("_initialize"); initFn != nil {
		if _, err := initFn.Call(ctx); err != nil {
			mod.Close(ctx)
			rt.Close(ctx)
			return nil, 0, fmt.Errorf("_initialize: %w", err)
		}
	}

	for _, name := range []string{"agent_init", "agent_tick", "agent_checkpoint", "agent_checkpoint_ptr", "agent_resume"} {
		if mod.ExportedFunction(name) == nil {
			mod.Close(ctx)
			rt.Close(ctx)
			return nil, 0, fmt.Errorf("missing required export: %s", name)
		}
	}

	if _, err := mod.ExportedFunction("agent_init").Call(ctx); err != nil {
		mod.Close(ctx)
		rt.Close(ctx)
		return nil, 0, fmt.Errorf("agent_init: %w", err)
	}

	var replayEngine *replay.Engine
	if cfg.Verify {
		replayEngine = replay.NewEngine(logger)
	}

	env := &simEnv{
		rt:             rt,
		mod:            mod,
		el:             el,
		wasmBytes:      wasmBytes,
		capManifest:    capManifest,
		replayEngine:   replayEngine,
		pricePerSecond: pricePerSecond,
		logger:         logger,
	}
	return env, budgetMicrocents, nil
}

// registerHostcalls sets up either deterministic or live hostcalls.
func registerHostcalls(
	ctx context.Context,
	cfg Config,
	rt wazero.Runtime,
	capManifest *manifest.CapabilityManifest,
	el *eventlog.EventLog,
	logger *slog.Logger,
) error {
	if cfg.Deterministic {
		clockStart := cfg.ClockStart
		if clockStart == 0 {
			clockStart = 1_000_000_000
		}
		clockDelta := cfg.ClockDelta
		if clockDelta == 0 {
			clockDelta = 1_000_000_000
		}
		dhc := newDeterministicHostcalls(clockStart, clockDelta, cfg.RandSeed, el, logger)
		if err := dhc.registerHostModule(ctx, rt, capManifest); err != nil {
			return fmt.Errorf("register deterministic hostcalls: %w", err)
		}
		return nil
	}
	reg := hostcall.NewRegistry(logger, el)
	if err := reg.RegisterHostModule(ctx, rt, capManifest); err != nil {
		return fmt.Errorf("register hostcalls: %w", err)
	}
	return nil
}

// runTickLoop drives the agent tick loop and returns the remaining budget.
func runTickLoop(ctx context.Context, env *simEnv, cfg Config, result *Result, budgetMicrocents int64) int64 {
	maxTicks := cfg.Ticks
	if maxTicks == 0 {
		maxTicks = 1<<31 - 1
	}

	for i := 0; i < maxTicks; i++ {
		if budgetMicrocents <= 0 {
			env.logger.Info("Budget exhausted, stopping", "tick", i)
			break
		}

		remaining, ok := executeTick(ctx, env, result, uint64(i+1), budgetMicrocents)
		budgetMicrocents = remaining
		if !ok {
			break
		}
	}
	return budgetMicrocents
}

// executeTick runs a single tick, captures state, and optionally verifies replay.
// Returns updated budget and false if the loop should stop.
func executeTick(ctx context.Context, env *simEnv, result *Result, tickNum uint64, budgetMicrocents int64) (int64, bool) {
	preState, err := captureState(ctx, env.mod)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("tick %d pre-state: %v", tickNum, err))
		return budgetMicrocents, false
	}

	env.el.BeginTick(tickNum)

	tickCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	start := time.Now()
	_, tickErr := env.mod.ExportedFunction("agent_tick").Call(tickCtx)
	elapsed := time.Since(start)
	cancel()

	sealed := env.el.SealTick()

	if tickErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("tick %d: %v", tickNum, tickErr))
		return budgetMicrocents, false
	}

	postState, err := captureState(ctx, env.mod)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("tick %d post-state: %v", tickNum, err))
		return budgetMicrocents, false
	}

	costMicrocents := elapsed.Nanoseconds() * env.pricePerSecond / 1_000_000_000
	budgetMicrocents -= costMicrocents
	result.TicksExecuted++

	if env.replayEngine != nil && sealed != nil {
		verifyTick(ctx, env, result, tickNum, preState, postState, sealed)
	}

	observationCount := 0
	if sealed != nil {
		observationCount = len(sealed.Entries)
	}
	env.logger.Info("Tick completed",
		"tick", tickNum,
		"duration_us", elapsed.Microseconds(),
		"cost", budget.Format(costMicrocents),
		"observations", observationCount,
	)
	return budgetMicrocents, true
}

// verifyTick replays a single tick and checks for determinism.
func verifyTick(ctx context.Context, env *simEnv, result *Result, tickNum uint64, preState, postState []byte, sealed *eventlog.TickLog) {
	rr := env.replayEngine.ReplayTick(ctx, env.wasmBytes, env.capManifest, preState, sealed, nil)
	if rr.Error != nil {
		result.ReplayFailed++
		result.Errors = append(result.Errors, fmt.Sprintf("tick %d replay: %v", tickNum, rr.Error))
		return
	}
	replayedHash := sha256.Sum256(rr.ReplayedState)
	expectedHash := sha256.Sum256(postState)
	if replayedHash == expectedHash {
		result.ReplayVerified++
	} else {
		result.ReplayFailed++
		result.Errors = append(result.Errors, fmt.Sprintf("tick %d replay: state divergence", tickNum))
	}
}

// finalize performs final checkpoint and round-trip verification.
func finalize(ctx context.Context, env *simEnv, result *Result) {
	finalState, err := captureState(ctx, env.mod)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("final checkpoint: %v", err))
		return
	}
	result.CheckpointSize = len(finalState)

	resumedState, rtErr := verifyCheckpointRoundTrip(ctx, env.mod, finalState)
	if rtErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("checkpoint round-trip: %v", rtErr))
	} else if sha256.Sum256(resumedState) != sha256.Sum256(finalState) {
		result.Errors = append(result.Errors, "checkpoint round-trip: state mismatch after resume")
	}
}

// PrintSummary writes the simulation result to the logger.
func PrintSummary(r *Result, logger *slog.Logger) {
	logger.Info("Simulation complete",
		"ticks_executed", r.TicksExecuted,
		"budget_consumed", budget.Format(r.BudgetConsumed),
		"budget_remaining", budget.Format(r.FinalBudget),
		"checkpoint_bytes", r.CheckpointSize,
		"replay_verified", r.ReplayVerified,
		"replay_failed", r.ReplayFailed,
		"errors", len(r.Errors),
	)
	for _, e := range r.Errors {
		logger.Error("Simulation error", "detail", e)
	}
}

func loadManifest(cfg Config) []byte {
	mPath := cfg.ManifestPath
	if mPath == "" && cfg.WASMPath != "" {
		mPath = cfg.WASMPath[:len(cfg.WASMPath)-len(".wasm")] + ".manifest.json"
	}
	data, err := os.ReadFile(mPath)
	if err != nil {
		return []byte("{}")
	}
	return data
}

func captureState(ctx context.Context, mod api.Module) ([]byte, error) {
	return wasmutil.CaptureState(ctx, mod)
}

func verifyCheckpointRoundTrip(ctx context.Context, mod api.Module, state []byte) ([]byte, error) {
	if len(state) == 0 {
		return []byte{}, nil
	}
	if err := wasmutil.ResumeAgent(ctx, mod, state); err != nil {
		return nil, err
	}
	return wasmutil.CaptureState(ctx, mod)
}
