// Package replay implements deterministic single-tick replay verification.
// Given a checkpoint, an event log, and the expected resulting state, the replay
// engine re-executes a tick with recorded observation values and verifies the
// agent produces identical output — closing the loop on CM-4 and EI-3.
package replay

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/simonovic86/igor/pkg/manifest"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Result describes the outcome of replaying a single tick.
type Result struct {
	// Verified is true when the replayed state matches the expected state.
	Verified bool

	// TickNumber is the tick that was replayed.
	TickNumber uint64

	// ReplayedState is the checkpoint state produced by the replay execution.
	ReplayedState []byte

	// ExpectedState is the checkpoint state that the replay was compared against.
	ExpectedState []byte

	// FirstDiffByte is the byte offset of the first difference, or -1 if matched.
	FirstDiffByte int

	// Error is set when replay could not complete (setup failure, WASM trap, etc.).
	// A non-nil Error means Verified is meaningless.
	Error error
}

// Engine performs single-tick replay verification.
type Engine struct {
	logger *slog.Logger
}

// NewEngine creates a replay engine.
func NewEngine(logger *slog.Logger) *Engine {
	return &Engine{logger: logger}
}

// ReplayTick performs single-tick replay verification.
//
// It creates an isolated wazero runtime, instantiates the WASM module with
// replay-mode hostcalls that return recorded observation values, resumes from
// the given checkpoint state, executes one tick, and compares the resulting
// state against expectedState.
func (e *Engine) ReplayTick(
	ctx context.Context,
	wasmBytes []byte,
	capManifest *manifest.CapabilityManifest,
	state []byte,
	tickLog *eventlog.TickLog,
	expectedState []byte,
) *Result {
	result := &Result{
		TickNumber:    tickLog.TickNumber,
		ExpectedState: expectedState,
		FirstDiffByte: -1,
	}

	// Create isolated wazero runtime (64MB memory limit, same as production)
	config := wazero.NewRuntimeConfig().
		WithMemoryLimitPages(1024).
		WithCloseOnContextDone(true)
	rt := wazero.NewRuntimeWithConfig(ctx, config)
	defer rt.Close(ctx)

	// Instantiate WASI
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	// Create entry iterator from tick log
	iter := &entryIterator{entries: tickLog.Entries}
	repErr := &replayError{}

	// Register replay hostcalls
	if err := registerReplayHostModule(ctx, rt, capManifest, iter, repErr); err != nil {
		result.Error = fmt.Errorf("replay: register host module: %w", err)
		return result
	}

	// Compile and instantiate WASM module
	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		result.Error = fmt.Errorf("replay: compile WASM: %w", err)
		return result
	}

	mod, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
		WithName("replay").
		WithStartFunctions())
	if err != nil {
		result.Error = fmt.Errorf("replay: instantiate module: %w", err)
		return result
	}
	defer mod.Close(ctx)

	// Run _initialize for TinyGo reactor modules
	if initFn := mod.ExportedFunction("_initialize"); initFn != nil {
		if _, err := initFn.Call(ctx); err != nil {
			result.Error = fmt.Errorf("replay: _initialize: %w", err)
			return result
		}
	}

	// Resume from checkpoint state
	if err := replayResume(ctx, mod, state); err != nil {
		result.Error = fmt.Errorf("replay: resume: %w", err)
		return result
	}

	// Execute agent_tick
	tickFn := mod.ExportedFunction("agent_tick")
	if tickFn == nil {
		result.Error = fmt.Errorf("replay: agent_tick not exported")
		return result
	}
	if _, err := tickFn.Call(ctx); err != nil {
		result.Error = fmt.Errorf("replay: agent_tick: %w", err)
		return result
	}

	// Check for replay hostcall errors
	if repErr.err != nil {
		result.Error = fmt.Errorf("replay: hostcall error: %w", repErr.err)
		return result
	}

	// Check that all entries were consumed
	if iter.remaining() > 0 {
		result.Error = fmt.Errorf("replay: %d unconsumed event log entries", iter.remaining())
		return result
	}

	// Checkpoint the replayed state
	replayedState, err := replayCheckpoint(ctx, mod)
	if err != nil {
		result.Error = fmt.Errorf("replay: checkpoint: %w", err)
		return result
	}
	result.ReplayedState = replayedState

	// Compare against expected state
	result.Verified = bytesEqual(replayedState, expectedState)
	if !result.Verified {
		result.FirstDiffByte = firstDiff(replayedState, expectedState)
		e.logger.Warn("Replay divergence detected",
			"tick", tickLog.TickNumber,
			"replayed_len", len(replayedState),
			"expected_len", len(expectedState),
			"first_diff_byte", result.FirstDiffByte,
		)
	} else {
		e.logger.Info("Replay verified",
			"tick", tickLog.TickNumber,
			"state_bytes", len(replayedState),
		)
	}

	return result
}

// entryIterator reads sequentially from a TickLog's entries.
type entryIterator struct {
	entries []eventlog.Entry
	pos     int
}

// next returns the next entry, or an error if exhausted or mismatched.
func (it *entryIterator) next(expectedID eventlog.HostcallID) (eventlog.Entry, error) {
	if it.pos >= len(it.entries) {
		return eventlog.Entry{}, fmt.Errorf(
			"event log exhausted at position %d, expected hostcall %d",
			it.pos, expectedID,
		)
	}
	entry := it.entries[it.pos]
	if entry.HostcallID != expectedID {
		return eventlog.Entry{}, fmt.Errorf(
			"hostcall mismatch at position %d: expected %d, got %d",
			it.pos, expectedID, entry.HostcallID,
		)
	}
	it.pos++
	return entry, nil
}

// remaining returns the number of unconsumed entries.
func (it *entryIterator) remaining() int {
	return len(it.entries) - it.pos
}

// replayError accumulates hostcall errors that cannot be returned directly
// from wazero callback functions (fixed WASM import signatures).
type replayError struct {
	err error
}

// registerReplayHostModule registers the "igor" host module with replay-mode
// hostcall implementations that consume entries from the iterator.
func registerReplayHostModule(
	ctx context.Context,
	rt wazero.Runtime,
	m *manifest.CapabilityManifest,
	iter *entryIterator,
	repErr *replayError,
) error {
	builder := rt.NewHostModuleBuilder("igor")
	registered := 0

	if m.Has("clock") {
		registerReplayClock(builder, iter, repErr)
		registered++
	}

	if m.Has("rand") {
		registerReplayRand(builder, iter, repErr)
		registered++
	}

	if m.Has("log") {
		registerReplayLog(builder, iter, repErr)
		registered++
	}

	if registered == 0 {
		return nil
	}

	_, err := builder.Instantiate(ctx)
	return err
}

// registerReplayClock registers clock_now that returns the recorded timestamp.
func registerReplayClock(
	builder wazero.HostModuleBuilder,
	iter *entryIterator,
	repErr *replayError,
) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int64 {
			entry, err := iter.next(eventlog.ClockNow)
			if err != nil {
				repErr.err = err
				return 0
			}
			if len(entry.Payload) != 8 {
				repErr.err = fmt.Errorf(
					"clock_now payload length %d, expected 8", len(entry.Payload),
				)
				return 0
			}
			return int64(binary.LittleEndian.Uint64(entry.Payload))
		}).
		Export("clock_now")
}

// registerReplayRand registers rand_bytes that writes recorded random bytes.
func registerReplayRand(
	builder wazero.HostModuleBuilder,
	iter *entryIterator,
	repErr *replayError,
) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, ptr, length uint32) int32 {
			entry, err := iter.next(eventlog.RandBytes)
			if err != nil {
				repErr.err = err
				return -1
			}
			if uint32(len(entry.Payload)) != length {
				repErr.err = fmt.Errorf(
					"rand_bytes payload length %d, expected %d",
					len(entry.Payload), length,
				)
				return -1
			}
			if !m.Memory().Write(ptr, entry.Payload) {
				repErr.err = fmt.Errorf("rand_bytes memory write failed")
				return -4
			}
			return 0
		}).
		Export("rand_bytes")
}

// registerReplayLog registers log_emit that silently consumes the recorded entry.
func registerReplayLog(
	builder wazero.HostModuleBuilder,
	iter *entryIterator,
	repErr *replayError,
) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, _ api.Module, _, _ uint32) {
			_, err := iter.next(eventlog.LogEmit)
			if err != nil {
				repErr.err = err
			}
		}).
		Export("log_emit")
}

// replayResume restores agent state in the replay module.
func replayResume(ctx context.Context, mod api.Module, state []byte) error {
	fn := mod.ExportedFunction("agent_resume")
	if fn == nil {
		return fmt.Errorf("agent_resume not exported")
	}

	if len(state) == 0 {
		_, err := fn.Call(ctx, 0, 0)
		return err
	}

	malloc := mod.ExportedFunction("malloc")
	if malloc == nil {
		return fmt.Errorf("malloc not exported (required for agent_resume)")
	}

	results, err := malloc.Call(ctx, uint64(len(state)))
	if err != nil {
		return fmt.Errorf("malloc: %w", err)
	}
	ptr := uint32(results[0])

	if !mod.Memory().Write(ptr, state) {
		return fmt.Errorf("failed to write state to WASM memory")
	}

	_, err = fn.Call(ctx, uint64(ptr), uint64(len(state)))
	return err
}

// replayCheckpoint extracts the agent's state from the replay module.
func replayCheckpoint(ctx context.Context, mod api.Module) ([]byte, error) {
	fnSize := mod.ExportedFunction("agent_checkpoint")
	if fnSize == nil {
		return nil, fmt.Errorf("agent_checkpoint not exported")
	}

	sizeResults, err := fnSize.Call(ctx)
	if err != nil {
		return nil, fmt.Errorf("agent_checkpoint: %w", err)
	}
	size := uint32(sizeResults[0])
	if size == 0 {
		return []byte{}, nil
	}

	fnPtr := mod.ExportedFunction("agent_checkpoint_ptr")
	if fnPtr == nil {
		return nil, fmt.Errorf("agent_checkpoint_ptr not exported")
	}

	ptrResults, err := fnPtr.Call(ctx)
	if err != nil {
		return nil, fmt.Errorf("agent_checkpoint_ptr: %w", err)
	}
	ptr := uint32(ptrResults[0])

	data, ok := mod.Memory().Read(ptr, size)
	if !ok {
		return nil, fmt.Errorf("failed to read checkpoint from memory")
	}

	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

// bytesEqual compares two byte slices for equality.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// firstDiff returns the index of the first differing byte, or -1 if equal.
func firstDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return n
	}
	return -1
}
