package agent

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"time"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/simonovic86/igor/internal/hostcall"
	"github.com/simonovic86/igor/internal/runtime"
	"github.com/simonovic86/igor/internal/storage"
	"github.com/simonovic86/igor/internal/wasmutil"
	"github.com/simonovic86/igor/pkg/budget"
	"github.com/simonovic86/igor/pkg/manifest"
	"github.com/simonovic86/igor/pkg/receipt"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const (
	checkpointVersion   byte = 0x02
	checkpointHeaderLen int  = 57 // 1 (version) + 8 (budget) + 8 (pricePerSecond) + 8 (tickNumber) + 32 (wasmHash)

	// DefaultReplayWindowSize is the number of recent tick snapshots retained
	// for sliding replay verification (CM-4).
	DefaultReplayWindowSize = 16
)

// TickSnapshot holds replay verification data for a single tick.
// Stored in the Instance's replay window for CM-4 sliding verification.
// PostStateHash stores only the SHA-256 hash of the post-tick state to halve
// snapshot memory usage (IMPROVEMENTS #2). The full pre-tick state is retained
// because it is needed as replay input.
type TickSnapshot struct {
	TickNumber    uint64
	PreState      []byte
	PostStateHash [32]byte
	TickLog       *eventlog.TickLog
}

// observationScore returns the number of observation entries in this tick.
// A higher score means the tick is more valuable for replay verification.
func (snap TickSnapshot) observationScore() int {
	if snap.TickLog == nil {
		return 0
	}
	return len(snap.TickLog.Entries)
}

// Instance represents a running agent instance.
type Instance struct {
	AgentID        string
	WASMBytes      []byte   // Raw WASM binary (retained for replay verification)
	WASMHash       [32]byte // SHA-256 of WASMBytes, stored in checkpoint header for integrity
	Compiled       wazero.CompiledModule
	Module         api.Module
	Engine         *runtime.Engine
	Storage        storage.Provider
	State          []byte
	Budget         int64 // Remaining budget in microcents (1 currency unit = 1,000,000 microcents)
	PricePerSecond int64 // Cost per second in microcents
	Manifest       *manifest.CapabilityManifest
	FullManifest   *manifest.Manifest // Full manifest with ResourceLimits and MigrationPolicy
	EventLog       *eventlog.EventLog
	TickNumber     uint64
	logger         *slog.Logger

	// Replay verification state: sliding window of recent tick snapshots (CM-4).
	ReplayWindow    []TickSnapshot // Recent tick snapshots for verification
	replayWindowMax int            // Maximum snapshots retained (0 = use DefaultReplayWindowSize)

	// Receipt tracking (Phase 4: Economics)
	Receipts        []receipt.Receipt  // Accumulated payment receipts
	lastReceiptTick uint64             // Tick number of the last receipt's epoch end
	epochCost       int64              // Accumulated cost since last receipt
	signingKey      ed25519.PrivateKey // Node's signing key; nil = receipts disabled
	nodeID          string             // Node's peer ID string
}

// walletStateRef is an indirection layer that lets wallet hostcall closures
// reference the Instance before it is fully constructed. The ref is populated
// after Instance creation; wallet hostcalls are only invoked during agent_tick.
type walletStateRef struct {
	instance *Instance
}

func (w *walletStateRef) GetBudget() int64 { return w.instance.Budget }
func (w *walletStateRef) GetReceiptCount() int {
	return len(w.instance.Receipts)
}
func (w *walletStateRef) GetReceiptBytes(index int) ([]byte, error) {
	return w.instance.GetReceiptBytes(index)
}

// GetBudget returns the current budget (implements hostcall.WalletState).
func (i *Instance) GetBudget() int64 {
	return i.Budget
}

// GetReceiptCount returns the number of receipts (implements hostcall.WalletState).
func (i *Instance) GetReceiptCount() int {
	return len(i.Receipts)
}

// GetReceiptBytes returns the serialized receipt at the given index.
func (i *Instance) GetReceiptBytes(index int) ([]byte, error) {
	if index < 0 || index >= len(i.Receipts) {
		return nil, fmt.Errorf("receipt index %d out of range [0, %d)", index, len(i.Receipts))
	}
	return i.Receipts[index].MarshalBinary(), nil
}

// CreateReceipt creates and signs a receipt for the current checkpoint epoch.
// Called after each successful checkpoint save. No-op if signing key is nil.
func (i *Instance) CreateReceipt() error {
	if i.signingKey == nil {
		return nil
	}
	r := receipt.Receipt{
		AgentID:        i.AgentID,
		NodeID:         i.nodeID,
		EpochStart:     i.lastReceiptTick + 1,
		EpochEnd:       i.TickNumber,
		CostMicrocents: i.epochCost,
		BudgetAfter:    i.Budget,
		Timestamp:      time.Now().UnixNano(),
	}
	if err := r.Sign(i.signingKey); err != nil {
		return fmt.Errorf("sign receipt: %w", err)
	}
	i.Receipts = append(i.Receipts, r)
	i.lastReceiptTick = i.TickNumber
	i.epochCost = 0
	return nil
}

// LoadAgent loads and compiles a WASM agent from a file path.
// manifestData is the JSON capability manifest; nil or empty means no capabilities.
// signingKey and nodeID enable payment receipt signing; pass nil/empty to disable.
func LoadAgent(
	ctx context.Context,
	engine *runtime.Engine,
	wasmPath string,
	agentID string,
	storageProvider storage.Provider,
	budgetVal int64,
	pricePerSecond int64,
	manifestData []byte,
	signingKey ed25519.PrivateKey,
	nodeID string,
	logger *slog.Logger,
) (*Instance, error) {
	logger.Info("Loading agent", "agent_id", agentID, "path", wasmPath)

	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read WASM file: %w", err)
	}

	return loadAgent(ctx, engine, wasmBytes, agentID, storageProvider, budgetVal, pricePerSecond, manifestData, signingKey, nodeID, logger)
}

// LoadAgentFromBytes loads and compiles a WASM agent from raw bytes.
// Used by the migration service to avoid writing WASM to a temporary file.
// signingKey and nodeID enable payment receipt signing; pass nil/empty to disable.
func LoadAgentFromBytes(
	ctx context.Context,
	engine *runtime.Engine,
	wasmBytes []byte,
	agentID string,
	storageProvider storage.Provider,
	budgetVal int64,
	pricePerSecond int64,
	manifestData []byte,
	signingKey ed25519.PrivateKey,
	nodeID string,
	logger *slog.Logger,
) (*Instance, error) {
	return loadAgent(ctx, engine, wasmBytes, agentID, storageProvider, budgetVal, pricePerSecond, manifestData, signingKey, nodeID, logger)
}

func loadAgent(
	ctx context.Context,
	engine *runtime.Engine,
	wasmBytes []byte,
	agentID string,
	storageProvider storage.Provider,
	budgetVal int64,
	pricePerSecond int64,
	manifestData []byte,
	signingKey ed25519.PrivateKey,
	nodeID string,
	logger *slog.Logger,
) (*Instance, error) {
	fullManifest, err := manifest.ParseManifest(manifestData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	capManifest := fullManifest.Capabilities

	if err := manifest.ValidateAgainstNode(capManifest, manifest.NodeCapabilities); err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	// Validate resource limits: reject agents that require more memory than the node provides.
	if fullManifest.ResourceLimits.MaxMemoryBytes > 0 &&
		fullManifest.ResourceLimits.MaxMemoryBytes > manifest.DefaultMaxMemoryBytes {
		return nil, fmt.Errorf("agent requires %d bytes memory, node provides %d",
			fullManifest.ResourceLimits.MaxMemoryBytes, manifest.DefaultMaxMemoryBytes)
	}

	logger.Info("Capability manifest loaded",
		"agent_id", agentID,
		"capabilities", capManifest.Names(),
	)

	el := eventlog.NewEventLog(eventlog.DefaultMaxTicks)

	// Create a wallet state ref that will be populated after the Instance is created.
	// The wallet hostcall closures capture this ref; it is only dereferenced during
	// agent_tick (not during loading), so the nil instance is safe at this point.
	wsRef := &walletStateRef{}

	registry := hostcall.NewRegistry(logger, el)
	registry.SetWalletState(wsRef)
	if err := registry.RegisterHostModule(ctx, engine.Runtime(), capManifest); err != nil {
		return nil, fmt.Errorf("failed to register host module: %w", err)
	}

	compiled, err := engine.CompileWASMBytes(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to compile WASM: %w", err)
	}

	module, err := engine.InstantiateModule(ctx, compiled, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate module: %w", err)
	}

	if initFn := module.ExportedFunction("_initialize"); initFn != nil {
		if _, err := initFn.Call(ctx); err != nil {
			module.Close(ctx)
			return nil, fmt.Errorf("_initialize failed: %w", err)
		}
	}

	instance := &Instance{
		AgentID:        agentID,
		WASMBytes:      wasmBytes,
		WASMHash:       sha256.Sum256(wasmBytes),
		Compiled:       compiled,
		Module:         module,
		Engine:         engine,
		Storage:        storageProvider,
		State:          nil,
		Budget:         budgetVal,
		PricePerSecond: pricePerSecond,
		Manifest:       capManifest,
		FullManifest:   fullManifest,
		EventLog:       el,
		TickNumber:     0,
		logger:         logger,
		signingKey:     signingKey,
		nodeID:         nodeID,
	}

	// Now that the instance exists, wire the wallet state ref so wallet
	// hostcalls can access budget and receipts during agent_tick.
	wsRef.instance = instance

	if err := instance.verifyExports(); err != nil {
		return nil, fmt.Errorf("agent lifecycle validation failed: %w", err)
	}

	logger.Info("Agent loaded successfully", "agent_id", agentID)
	return instance, nil
}

// verifyExports ensures the agent exports required lifecycle functions.
func (i *Instance) verifyExports() error {
	required := []string{"agent_init", "agent_tick", "agent_checkpoint", "agent_checkpoint_ptr", "agent_resume"}
	for _, name := range required {
		fn := i.Module.ExportedFunction(name)
		if fn == nil {
			return fmt.Errorf("missing required export: %s", name)
		}
	}
	return nil
}

// Init calls the agent's init function.
func (i *Instance) Init(ctx context.Context) error {
	i.logger.Info("Calling agent init", "agent_id", i.AgentID)

	fn := i.Module.ExportedFunction("agent_init")
	if fn == nil {
		return fmt.Errorf("agent_init function not found")
	}

	_, err := fn.Call(ctx)
	if err != nil {
		return fmt.Errorf("agent_init failed: %w", err)
	}

	i.logger.Info("Agent initialized", "agent_id", i.AgentID)
	return nil
}

// Tick executes one tick of the agent with a timeout and meters execution cost.
// Per CE-3, the event log records all observation hostcall return values.
// Pre-tick and post-tick state are captured for replay verification (CM-4).
// Returns hasMoreWork (true if agent signaled more work pending) and any error.
func (i *Instance) Tick(ctx context.Context) (bool, error) {
	// Check budget before execution
	if i.Budget <= 0 {
		return false, fmt.Errorf("budget exhausted: %s", budget.Format(i.Budget))
	}

	// Capture pre-tick state for replay verification
	preState, err := i.captureState(ctx)
	if err != nil {
		return false, fmt.Errorf("pre-tick checkpoint failed: %w", err)
	}

	// Advance tick counter and begin event log recording
	i.TickNumber++
	i.EventLog.BeginTick(i.TickNumber)

	// Enforce tick timeout
	tickCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	fn := i.Module.ExportedFunction("agent_tick")
	if fn == nil {
		return false, fmt.Errorf("agent_tick function not found")
	}

	start := time.Now()
	results, tickErr := fn.Call(tickCtx)
	elapsed := time.Since(start)

	// Seal the event log regardless of tick success/failure
	sealed := i.EventLog.SealTick()

	if tickErr != nil {
		return false, fmt.Errorf("agent_tick failed: %w", tickErr)
	}

	// Read adaptive tick hint from agent return value.
	// Legacy agents (void agent_tick) return no results; treat as "no more work".
	var hasMoreWork bool
	if len(results) > 0 {
		hasMoreWork = results[0] != 0
	}

	// Capture post-tick state for replay verification
	postState, err := i.captureState(ctx)
	if err != nil {
		return false, fmt.Errorf("post-tick checkpoint failed: %w", err)
	}

	// Store replay verification snapshot in sliding window.
	// Only the hash of the post-state is retained (IMPROVEMENTS #2).
	i.ReplayWindow = append(i.ReplayWindow, TickSnapshot{
		TickNumber:    i.TickNumber,
		PreState:      preState,
		PostStateHash: sha256.Sum256(postState),
		TickLog:       sealed,
	})
	maxSnaps := i.replayWindowMax
	if maxSnaps <= 0 {
		maxSnaps = DefaultReplayWindowSize
	}
	if len(i.ReplayWindow) > maxSnaps {
		// Weighted eviction: drop the snapshot with the lowest observation
		// score. Among ties, prefer evicting the oldest (lowest index).
		// Never evict the most recently added snapshot (last element).
		evictIdx := 0
		evictScore := i.ReplayWindow[0].observationScore()
		for j := 1; j < len(i.ReplayWindow)-1; j++ {
			score := i.ReplayWindow[j].observationScore()
			if score < evictScore {
				evictScore = score
				evictIdx = j
			}
		}
		i.ReplayWindow = append(i.ReplayWindow[:evictIdx], i.ReplayWindow[evictIdx+1:]...)
	}

	// Calculate and deduct execution cost (nanosecond precision, no float, no truncation).
	// Guard against int64 overflow: if nanos * price would overflow, cap cost at remaining budget.
	var costMicrocents int64
	nanos := elapsed.Nanoseconds()
	if i.PricePerSecond > 0 && nanos > math.MaxInt64/i.PricePerSecond {
		costMicrocents = i.Budget
	} else {
		costMicrocents = nanos * i.PricePerSecond / 1_000_000_000
	}
	i.Budget -= costMicrocents
	i.epochCost += costMicrocents

	observationCount := 0
	if sealed != nil {
		observationCount = len(sealed.Entries)
	}

	i.logger.Info("Tick completed",
		"agent_id", i.AgentID,
		"tick", i.TickNumber,
		"duration_ms", elapsed.Milliseconds(),
		"cost", budget.Format(costMicrocents),
		"budget_remaining", budget.Format(i.Budget),
		"observations", observationCount,
		"has_more_work", hasMoreWork,
	)

	return hasMoreWork, nil
}

// captureState extracts the agent's current state via checkpoint exports.
func (i *Instance) captureState(ctx context.Context) ([]byte, error) {
	return wasmutil.CaptureState(ctx, i.Module)
}

// Checkpoint saves the agent's state.
func (i *Instance) Checkpoint(ctx context.Context) ([]byte, error) {
	i.logger.Info("Checkpointing agent", "agent_id", i.AgentID)

	// Call agent_checkpoint() to get size
	fnSize := i.Module.ExportedFunction("agent_checkpoint")
	if fnSize == nil {
		return nil, fmt.Errorf("agent_checkpoint function not found")
	}

	sizeResults, err := fnSize.Call(ctx)
	if err != nil {
		return nil, fmt.Errorf("agent_checkpoint failed: %w", err)
	}

	size := uint32(sizeResults[0])
	if size == 0 {
		i.logger.Info("Agent checkpoint empty", "agent_id", i.AgentID)
		return []byte{}, nil
	}

	// Call agent_checkpoint_ptr() to get pointer
	fnPtr := i.Module.ExportedFunction("agent_checkpoint_ptr")
	if fnPtr == nil {
		return nil, fmt.Errorf("agent_checkpoint_ptr function not found")
	}

	ptrResults, err := fnPtr.Call(ctx)
	if err != nil {
		return nil, fmt.Errorf("agent_checkpoint_ptr failed: %w", err)
	}

	ptr := uint32(ptrResults[0])

	// Read state from WASM memory
	state, ok := i.Module.Memory().Read(ptr, size)
	if !ok {
		return nil, fmt.Errorf("failed to read checkpoint state from memory")
	}

	i.State = state
	i.logger.Info("Agent checkpointed",
		"agent_id", i.AgentID,
		"state_bytes", len(state),
	)

	return state, nil
}

// Resume restores the agent from a checkpointed state.
func (i *Instance) Resume(ctx context.Context, state []byte) error {
	i.logger.Info("Resuming agent",
		"agent_id", i.AgentID,
		"state_bytes", len(state),
	)

	fn := i.Module.ExportedFunction("agent_resume")
	if fn == nil {
		return fmt.Errorf("agent_resume function not found")
	}

	if len(state) == 0 {
		i.logger.Info("Resuming with empty state", "agent_id", i.AgentID)
		_, err := fn.Call(ctx, 0, 0)
		if err != nil {
			return fmt.Errorf("agent_resume failed: %w", err)
		}
		return nil
	}

	// Allocate memory in WASM for state
	malloc := i.Module.ExportedFunction("malloc")
	if malloc == nil {
		return fmt.Errorf("malloc function not found (required for agent_resume)")
	}

	results, err := malloc.Call(ctx, uint64(len(state)))
	if err != nil {
		return fmt.Errorf("malloc failed: %w", err)
	}

	ptr := uint32(results[0])
	if ptr == 0 {
		return fmt.Errorf("malloc returned null pointer (out of memory)")
	}

	// Write state to WASM memory
	ok := i.Module.Memory().Write(ptr, state)
	if !ok {
		return fmt.Errorf("failed to write state to WASM memory")
	}

	// Call agent_resume(ptr, len)
	_, err = fn.Call(ctx, uint64(ptr), uint64(len(state)))
	if err != nil {
		return fmt.Errorf("agent_resume failed: %w", err)
	}

	i.State = state
	i.logger.Info("Agent resumed", "agent_id", i.AgentID)
	return nil
}

// SaveCheckpointToStorage checkpoints the agent and saves to storage provider.
// The checkpoint includes budget metadata, tick number, and agent state.
func (i *Instance) SaveCheckpointToStorage(ctx context.Context) error {
	// Checkpoint agent state
	state, err := i.Checkpoint(ctx)
	if err != nil {
		return fmt.Errorf("failed to checkpoint agent: %w", err)
	}

	// Format: [version:1][budget:8][price:8][tick:8][wasmHash:32][state:N]
	checkpoint := make([]byte, checkpointHeaderLen+len(state))
	checkpoint[0] = checkpointVersion
	binary.LittleEndian.PutUint64(checkpoint[1:9], uint64(i.Budget))
	binary.LittleEndian.PutUint64(checkpoint[9:17], uint64(i.PricePerSecond))
	binary.LittleEndian.PutUint64(checkpoint[17:25], i.TickNumber)
	copy(checkpoint[25:57], i.WASMHash[:])
	copy(checkpoint[57:], state)

	// Save to storage provider
	if err := i.Storage.SaveCheckpoint(ctx, i.AgentID, checkpoint); err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	// Create and persist payment receipt for this epoch (non-fatal on failure).
	if err := i.CreateReceipt(); err != nil {
		i.logger.Error("Failed to create receipt", "error", err)
	}
	if len(i.Receipts) > 0 {
		data := receipt.MarshalReceipts(i.Receipts)
		if err := i.Storage.SaveReceipts(ctx, i.AgentID, data); err != nil {
			i.logger.Error("Failed to save receipts", "error", err)
		}
	}

	return nil
}

// LoadCheckpointFromStorage loads checkpoint from storage and resumes agent.
// The checkpoint includes budget metadata, tick number, and agent state.
func (i *Instance) LoadCheckpointFromStorage(ctx context.Context) error {
	// Load from storage provider
	checkpoint, err := i.Storage.LoadCheckpoint(ctx, i.AgentID)
	if err != nil {
		if errors.Is(err, storage.ErrCheckpointNotFound) {
			// No checkpoint exists - this is normal for new agents
			i.logger.Info("No existing checkpoint found", "agent_id", i.AgentID)
			return nil
		}
		return fmt.Errorf("failed to load checkpoint: %w", err)
	}

	restoredBudget, restoredPrice, restoredTick, storedHash, state, err := ParseCheckpointHeader(checkpoint)
	if err != nil {
		return fmt.Errorf("invalid checkpoint: %w", err)
	}

	if storedHash != i.WASMHash {
		return fmt.Errorf("WASM hash mismatch: checkpoint was created by a different binary")
	}

	i.Budget = restoredBudget
	i.PricePerSecond = restoredPrice
	i.TickNumber = restoredTick

	i.logger.Info("Checkpoint restored",
		"agent_id", i.AgentID,
		"budget", budget.Format(restoredBudget),
		"price_per_second", budget.Format(restoredPrice),
		"tick_number", restoredTick,
	)

	// Load receipts from storage (non-fatal: missing receipts is normal for old checkpoints).
	receiptData, receiptErr := i.Storage.LoadReceipts(ctx, i.AgentID)
	if receiptErr == nil {
		receipts, parseErr := receipt.UnmarshalReceipts(receiptData)
		if parseErr != nil {
			i.logger.Warn("Failed to parse receipts", "error", parseErr)
		} else {
			i.Receipts = receipts
			if len(receipts) > 0 {
				i.lastReceiptTick = receipts[len(receipts)-1].EpochEnd
			}
			i.logger.Info("Receipts restored", "count", len(receipts))
		}
	}

	// Resume agent from state
	if err := i.Resume(ctx, state); err != nil {
		return fmt.Errorf("failed to resume agent: %w", err)
	}

	return nil
}

// ParseCheckpointHeader parses a checkpoint header.
// Returns budget, pricePerSecond, tickNumber, wasmHash, agentState, and any error.
func ParseCheckpointHeader(checkpoint []byte) (budgetVal int64, price int64, tick uint64, wasmHash [32]byte, state []byte, err error) {
	if len(checkpoint) < checkpointHeaderLen {
		return 0, 0, 0, [32]byte{}, nil, fmt.Errorf("checkpoint too short: %d bytes (need %d)", len(checkpoint), checkpointHeaderLen)
	}
	if checkpoint[0] != checkpointVersion {
		return 0, 0, 0, [32]byte{}, nil, fmt.Errorf("unsupported checkpoint version: %d", checkpoint[0])
	}
	budgetParsed := int64(binary.LittleEndian.Uint64(checkpoint[1:9]))
	priceParsed := int64(binary.LittleEndian.Uint64(checkpoint[9:17]))
	if budgetParsed < 0 {
		return 0, 0, 0, [32]byte{}, nil, fmt.Errorf("checkpoint contains negative budget: %d", budgetParsed)
	}
	if priceParsed < 0 {
		return 0, 0, 0, [32]byte{}, nil, fmt.Errorf("checkpoint contains negative price: %d", priceParsed)
	}
	var hash [32]byte
	copy(hash[:], checkpoint[25:57])
	return budgetParsed,
		priceParsed,
		binary.LittleEndian.Uint64(checkpoint[17:25]),
		hash,
		checkpoint[checkpointHeaderLen:],
		nil
}

// ExtractAgentState extracts the agent state portion from a checkpoint.
func ExtractAgentState(checkpoint []byte) ([]byte, error) {
	_, _, _, _, state, err := ParseCheckpointHeader(checkpoint)
	return state, err
}

// SetReplayWindowSize configures the maximum number of tick snapshots retained
// in the replay window. Must be called before the first Tick.
func (i *Instance) SetReplayWindowSize(n int) {
	i.replayWindowMax = n
}

// LatestSnapshot returns a copy of the most recent tick snapshot, or nil if no
// ticks have been executed. Returns a copy to avoid pointer invalidation when
// the replay window evicts entries.
func (i *Instance) LatestSnapshot() *TickSnapshot {
	if len(i.ReplayWindow) == 0 {
		return nil
	}
	snap := i.ReplayWindow[len(i.ReplayWindow)-1]
	return &snap
}

// Close releases agent resources.
func (i *Instance) Close(ctx context.Context) error {
	if i.Module != nil {
		return i.Module.Close(ctx)
	}
	return nil
}
