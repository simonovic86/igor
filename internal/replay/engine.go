// SPDX-License-Identifier: Apache-2.0

// Package replay implements deterministic single-tick replay verification.
// Given a checkpoint, an event log, and the expected resulting state, the replay
// engine re-executes a tick with recorded observation values and verifies the
// agent produces identical output — closing the loop on CM-4 and EI-3.
package replay

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log/slog"
	"time"

	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/simonovic86/igor/internal/wasmutil"
	"github.com/simonovic86/igor/pkg/manifest"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// replayTickTimeout aliases the shared tick timeout constant.
const replayTickTimeout = agent.DefaultTickTimeout

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

	// Duration is the wall-clock time spent executing the replay.
	Duration time.Duration

	// Error is set when replay could not complete (setup failure, WASM trap, etc.).
	// A non-nil Error means Verified is meaningless.
	Error error
}

// Engine performs single-tick replay verification.
// A shared wazero CompilationCache avoids recompiling the same WASM binary
// on every call to ReplayTick (~300x speedup for repeated verification).
type Engine struct {
	logger *slog.Logger
	cache  wazero.CompilationCache
}

// NewEngine creates a replay engine with a shared compilation cache.
func NewEngine(logger *slog.Logger) *Engine {
	return &Engine{
		logger: logger,
		cache:  wazero.NewCompilationCache(),
	}
}

// Close releases the compilation cache resources.
func (e *Engine) Close(ctx context.Context) error {
	if e.cache != nil {
		return e.cache.Close(ctx)
	}
	return nil
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
	start := time.Now()
	result := &Result{
		TickNumber:    tickLog.TickNumber,
		ExpectedState: expectedState,
		FirstDiffByte: -1,
	}
	defer func() { result.Duration = time.Since(start) }()

	// Create isolated wazero runtime (64MB memory limit, same as production).
	// The shared compilation cache avoids recompiling the same WASM binary.
	config := wazero.NewRuntimeConfig().
		WithMemoryLimitPages(1024).
		WithCloseOnContextDone(true).
		WithCompilationCache(e.cache)
	rt := wazero.NewRuntimeWithConfig(ctx, config)
	defer rt.Close(ctx)

	// Instantiate WASI
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		result.Error = fmt.Errorf("replay: instantiate WASI: %w", err)
		return result
	}

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

	// Execute agent_tick with timeout matching production tick timeout (100ms).
	// Without this, a malicious or buggy agent could hang replay indefinitely.
	tickFn := mod.ExportedFunction("agent_tick")
	if tickFn == nil {
		result.Error = fmt.Errorf("replay: agent_tick not exported")
		return result
	}
	tickCtx, tickCancel := context.WithTimeout(ctx, replayTickTimeout)
	defer tickCancel()
	if _, err := tickFn.Call(tickCtx); err != nil {
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

	// Compare against expected state (when provided).
	// When expectedState is nil, the caller performs hash-based verification
	// on result.ReplayedState instead (IMPROVEMENTS #2).
	if expectedState != nil {
		result.Verified = bytes.Equal(replayedState, expectedState)
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

	if m.Has("wallet") {
		registerReplayWallet(builder, iter, repErr)
		registered++
	}

	if m.Has("pricing") {
		registerReplayPricing(builder, iter, repErr)
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

// registerReplayWallet registers wallet_balance, wallet_receipt_count, and
// wallet_receipt that return recorded values from the event log.
func registerReplayWallet(
	builder wazero.HostModuleBuilder,
	iter *entryIterator,
	repErr *replayError,
) {
	// wallet_balance() -> i64
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int64 {
			entry, err := iter.next(eventlog.WalletBalance)
			if err != nil {
				repErr.err = err
				return 0
			}
			if len(entry.Payload) != 8 {
				repErr.err = fmt.Errorf(
					"wallet_balance payload length %d, expected 8", len(entry.Payload),
				)
				return 0
			}
			return int64(binary.LittleEndian.Uint64(entry.Payload))
		}).
		Export("wallet_balance")

	// wallet_receipt_count() -> i32
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int32 {
			entry, err := iter.next(eventlog.WalletReceiptCount)
			if err != nil {
				repErr.err = err
				return 0
			}
			if len(entry.Payload) != 4 {
				repErr.err = fmt.Errorf(
					"wallet_receipt_count payload length %d, expected 4", len(entry.Payload),
				)
				return 0
			}
			return int32(binary.LittleEndian.Uint32(entry.Payload))
		}).
		Export("wallet_receipt_count")

	// wallet_receipt(index i32, buf_ptr i32, buf_len i32) -> i32
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, _ int32, ptr, length uint32) int32 {
			entry, err := iter.next(eventlog.WalletReceipt)
			if err != nil {
				repErr.err = err
				return -1
			}
			if uint32(len(entry.Payload)) > length {
				repErr.err = fmt.Errorf("wallet_receipt payload %d exceeds buffer %d",
					len(entry.Payload), length)
				return -4
			}
			if !m.Memory().Write(ptr, entry.Payload) {
				repErr.err = fmt.Errorf("wallet_receipt memory write failed")
				return -4
			}
			return int32(len(entry.Payload))
		}).
		Export("wallet_receipt")
}

// registerReplayPricing registers node_price that returns the recorded value.
func registerReplayPricing(
	builder wazero.HostModuleBuilder,
	iter *entryIterator,
	repErr *replayError,
) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int64 {
			entry, err := iter.next(eventlog.NodePrice)
			if err != nil {
				repErr.err = err
				return 0
			}
			if len(entry.Payload) != 8 {
				repErr.err = fmt.Errorf(
					"node_price payload length %d, expected 8", len(entry.Payload),
				)
				return 0
			}
			return int64(binary.LittleEndian.Uint64(entry.Payload))
		}).
		Export("node_price")
}

// replayResume restores agent state in the replay module.
func replayResume(ctx context.Context, mod api.Module, state []byte) error {
	return wasmutil.ResumeAgent(ctx, mod, state)
}

// replayCheckpoint extracts the agent's state from the replay module.
func replayCheckpoint(ctx context.Context, mod api.Module) ([]byte, error) {
	return wasmutil.CaptureState(ctx, mod)
}

// ChainSnapshot holds the data needed to replay a single tick within a chain.
// Defined in the replay package to avoid importing internal/agent.
type ChainSnapshot struct {
	TickNumber uint64
	TickLog    *eventlog.TickLog
}

// ChainResult describes the outcome of replaying a chain of consecutive ticks.
type ChainResult struct {
	Verified           bool
	TicksReplayed      int
	FirstTick          uint64
	LastTick           uint64
	FinalReplayedState []byte
	ExpectedFinalHash  [32]byte
	Duration           time.Duration
	Error              error
	FailedAtTick       uint64
}

// iteratorHolder allows chain replay hostcalls to reference the current tick's
// entry iterator. The pointer is swapped between ticks without re-registering
// the host module.
type iteratorHolder struct {
	iter *entryIterator
}

// ReplayChain replays N consecutive ticks in a single WASM instance and
// compares only the final state hash against expectedFinalHash. This catches
// accumulated drift that single-tick replay misses.
func (e *Engine) ReplayChain(
	ctx context.Context,
	wasmBytes []byte,
	capManifest *manifest.CapabilityManifest,
	initialState []byte,
	snapshots []ChainSnapshot,
	expectedFinalHash [32]byte,
) *ChainResult {
	start := time.Now()
	result := &ChainResult{
		ExpectedFinalHash: expectedFinalHash,
	}
	defer func() { result.Duration = time.Since(start) }()

	if len(snapshots) == 0 {
		result.Error = fmt.Errorf("replay chain: no snapshots provided")
		return result
	}

	result.FirstTick = snapshots[0].TickNumber
	result.LastTick = snapshots[len(snapshots)-1].TickNumber

	// Create a single wazero runtime for the entire chain.
	config := wazero.NewRuntimeConfig().
		WithMemoryLimitPages(1024).
		WithCloseOnContextDone(true).
		WithCompilationCache(e.cache)
	rt := wazero.NewRuntimeWithConfig(ctx, config)
	defer rt.Close(ctx)

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		result.Error = fmt.Errorf("replay chain: instantiate WASI: %w", err)
		return result
	}

	holder := &iteratorHolder{}
	repErr := &replayError{}

	if err := registerChainReplayHostModule(ctx, rt, capManifest, holder, repErr); err != nil {
		result.Error = fmt.Errorf("replay chain: register host module: %w", err)
		return result
	}

	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		result.Error = fmt.Errorf("replay chain: compile: %w", err)
		return result
	}

	mod, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
		WithName("replay-chain").
		WithStartFunctions())
	if err != nil {
		result.Error = fmt.Errorf("replay chain: instantiate: %w", err)
		return result
	}
	defer mod.Close(ctx)

	if initFn := mod.ExportedFunction("_initialize"); initFn != nil {
		if _, err := initFn.Call(ctx); err != nil {
			result.Error = fmt.Errorf("replay chain: _initialize: %w", err)
			return result
		}
	}

	if err := replayResume(ctx, mod, initialState); err != nil {
		result.Error = fmt.Errorf("replay chain: initial resume: %w", err)
		return result
	}

	tickFn := mod.ExportedFunction("agent_tick")
	if tickFn == nil {
		result.Error = fmt.Errorf("replay chain: agent_tick not exported")
		return result
	}

	for i, snap := range snapshots {
		if snap.TickLog == nil {
			holder.iter = &entryIterator{}
		} else {
			holder.iter = &entryIterator{entries: snap.TickLog.Entries}
		}
		repErr.err = nil

		tickCtx, tickCancel := context.WithTimeout(ctx, replayTickTimeout)
		if _, err := tickFn.Call(tickCtx); err != nil {
			tickCancel()
			result.Error = fmt.Errorf("replay chain: tick %d: %w", snap.TickNumber, err)
			result.FailedAtTick = snap.TickNumber
			return result
		}
		tickCancel()

		if repErr.err != nil {
			result.Error = fmt.Errorf("replay chain: tick %d hostcall: %w", snap.TickNumber, repErr.err)
			result.FailedAtTick = snap.TickNumber
			return result
		}

		if holder.iter.remaining() > 0 {
			result.Error = fmt.Errorf("replay chain: tick %d: %d unconsumed entries",
				snap.TickNumber, holder.iter.remaining())
			result.FailedAtTick = snap.TickNumber
			return result
		}

		result.TicksReplayed = i + 1
	}

	finalState, err := replayCheckpoint(ctx, mod)
	if err != nil {
		result.Error = fmt.Errorf("replay chain: final checkpoint: %w", err)
		return result
	}
	result.FinalReplayedState = finalState

	replayedHash := sha256.Sum256(finalState)
	result.Verified = replayedHash == expectedFinalHash

	if result.Verified {
		e.logger.Info("Chain replay verified",
			"ticks", result.TicksReplayed,
			"first_tick", result.FirstTick,
			"last_tick", result.LastTick,
		)
	} else {
		e.logger.Warn("Chain replay divergence detected",
			"ticks", result.TicksReplayed,
			"first_tick", result.FirstTick,
			"last_tick", result.LastTick,
		)
	}

	return result
}

// registerChainReplayHostModule registers replay hostcalls that read from an
// iteratorHolder, allowing the iterator to be swapped between ticks.
func registerChainReplayHostModule(
	ctx context.Context,
	rt wazero.Runtime,
	m *manifest.CapabilityManifest,
	holder *iteratorHolder,
	repErr *replayError,
) error {
	builder := rt.NewHostModuleBuilder("igor")
	registered := 0

	if m.Has("clock") {
		builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context) int64 {
				entry, err := holder.iter.next(eventlog.ClockNow)
				if err != nil {
					repErr.err = err
					return 0
				}
				if len(entry.Payload) != 8 {
					repErr.err = fmt.Errorf("clock_now payload length %d, expected 8", len(entry.Payload))
					return 0
				}
				return int64(binary.LittleEndian.Uint64(entry.Payload))
			}).
			Export("clock_now")
		registered++
	}

	if m.Has("rand") {
		builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, mod api.Module, ptr, length uint32) int32 {
				entry, err := holder.iter.next(eventlog.RandBytes)
				if err != nil {
					repErr.err = err
					return -1
				}
				if uint32(len(entry.Payload)) != length {
					repErr.err = fmt.Errorf("rand_bytes payload length %d, expected %d", len(entry.Payload), length)
					return -1
				}
				if !mod.Memory().Write(ptr, entry.Payload) {
					repErr.err = fmt.Errorf("rand_bytes memory write failed")
					return -4
				}
				return 0
			}).
			Export("rand_bytes")
		registered++
	}

	if m.Has("log") {
		builder.NewFunctionBuilder().
			WithFunc(func(_ context.Context, _ api.Module, _, _ uint32) {
				_, err := holder.iter.next(eventlog.LogEmit)
				if err != nil {
					repErr.err = err
				}
			}).
			Export("log_emit")
		registered++
	}

	if m.Has("wallet") {
		registerChainReplayWallet(builder, holder, repErr)
		registered++
	}

	if m.Has("pricing") {
		registerChainReplayPricing(builder, holder, repErr)
		registered++
	}

	if registered == 0 {
		return nil
	}

	_, err := builder.Instantiate(ctx)
	return err
}

// registerChainReplayWallet registers wallet replay hostcalls that read from
// the iteratorHolder, allowing the iterator to be swapped between ticks.
func registerChainReplayWallet(
	builder wazero.HostModuleBuilder,
	holder *iteratorHolder,
	repErr *replayError,
) {
	// wallet_balance() -> i64
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int64 {
			entry, err := holder.iter.next(eventlog.WalletBalance)
			if err != nil {
				repErr.err = err
				return 0
			}
			if len(entry.Payload) != 8 {
				repErr.err = fmt.Errorf("wallet_balance payload length %d, expected 8", len(entry.Payload))
				return 0
			}
			return int64(binary.LittleEndian.Uint64(entry.Payload))
		}).
		Export("wallet_balance")

	// wallet_receipt_count() -> i32
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int32 {
			entry, err := holder.iter.next(eventlog.WalletReceiptCount)
			if err != nil {
				repErr.err = err
				return 0
			}
			if len(entry.Payload) != 4 {
				repErr.err = fmt.Errorf("wallet_receipt_count payload length %d, expected 4", len(entry.Payload))
				return 0
			}
			return int32(binary.LittleEndian.Uint32(entry.Payload))
		}).
		Export("wallet_receipt_count")

	// wallet_receipt(index i32, buf_ptr i32, buf_len i32) -> i32
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, mod api.Module, _ int32, ptr, length uint32) int32 {
			entry, err := holder.iter.next(eventlog.WalletReceipt)
			if err != nil {
				repErr.err = err
				return -1
			}
			if uint32(len(entry.Payload)) > length {
				repErr.err = fmt.Errorf("wallet_receipt payload %d exceeds buffer %d", len(entry.Payload), length)
				return -4
			}
			if !mod.Memory().Write(ptr, entry.Payload) {
				repErr.err = fmt.Errorf("wallet_receipt memory write failed")
				return -4
			}
			return int32(len(entry.Payload))
		}).
		Export("wallet_receipt")
}

// registerChainReplayPricing registers node_price replay that reads from
// the iteratorHolder, allowing the iterator to be swapped between ticks.
func registerChainReplayPricing(
	builder wazero.HostModuleBuilder,
	holder *iteratorHolder,
	repErr *replayError,
) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int64 {
			entry, err := holder.iter.next(eventlog.NodePrice)
			if err != nil {
				repErr.err = err
				return 0
			}
			if len(entry.Payload) != 8 {
				repErr.err = fmt.Errorf("node_price payload length %d, expected 8", len(entry.Payload))
				return 0
			}
			return int64(binary.LittleEndian.Uint64(entry.Payload))
		}).
		Export("node_price")
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
