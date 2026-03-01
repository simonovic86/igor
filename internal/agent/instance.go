package agent

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/simonovic86/igor/internal/hostcall"
	"github.com/simonovic86/igor/internal/runtime"
	"github.com/simonovic86/igor/internal/storage"
	"github.com/simonovic86/igor/pkg/budget"
	"github.com/simonovic86/igor/pkg/manifest"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const (
	checkpointVersion   byte = 0x01
	checkpointHeaderLen int  = 17 // 1 (version) + 8 (budget) + 8 (pricePerSecond)
)

// Instance represents a running agent instance.
type Instance struct {
	AgentID        string
	WASMBytes      []byte // Raw WASM binary (retained for replay verification)
	Compiled       wazero.CompiledModule
	Module         api.Module
	Engine         *runtime.Engine
	Storage        storage.Provider
	State          []byte
	Budget         int64 // Remaining budget in microcents (1 currency unit = 1,000,000 microcents)
	PricePerSecond int64 // Cost per second in microcents
	Manifest       *manifest.CapabilityManifest
	EventLog       *eventlog.EventLog
	TickNumber     uint64
	logger         *slog.Logger

	// Replay verification state (captured each tick for CM-4 verification)
	PreTickState  []byte            // Agent state before the last tick
	PostTickState []byte            // Agent state after the last tick
	LastTickLog   *eventlog.TickLog // Sealed event log from the last tick
}

// LoadAgent loads and compiles a WASM agent from a file.
// manifestData is the JSON capability manifest; nil or empty means no capabilities.
func LoadAgent(
	ctx context.Context,
	engine *runtime.Engine,
	wasmPath string,
	agentID string,
	storageProvider storage.Provider,
	budget int64,
	pricePerSecond int64,
	manifestData []byte,
	logger *slog.Logger,
) (*Instance, error) {
	logger.Info("Loading agent", "agent_id", agentID, "path", wasmPath)

	// Parse capability manifest
	capManifest, err := manifest.ParseCapabilityManifest(manifestData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Validate manifest against node capabilities
	if err := manifest.ValidateAgainstNode(capManifest, manifest.NodeCapabilities); err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	logger.Info("Capability manifest loaded",
		"agent_id", agentID,
		"capabilities", capManifest.Names(),
	)

	// Create event log for observation recording
	el := eventlog.NewEventLog(eventlog.DefaultMaxTicks)

	// Register igor host module with declared capabilities (CE-1, CE-2)
	registry := hostcall.NewRegistry(logger, el)
	if err := registry.RegisterHostModule(ctx, engine.Runtime(), capManifest); err != nil {
		return nil, fmt.Errorf("failed to register host module: %w", err)
	}

	// Read and compile WASM module (bytes retained for replay verification)
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read WASM file: %w", err)
	}

	compiled, err := engine.CompileWASMBytes(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to compile WASM: %w", err)
	}

	// Instantiate module — if the agent imports from "igor" but the capability
	// was not declared, wazero will fail here with a clear import error (CM-3).
	module, err := engine.InstantiateModule(ctx, compiled, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate module: %w", err)
	}

	// Initialize the WASM module runtime. TinyGo agents export _initialize
	// (WASI reactor mode); standard Go agents may export _start.
	// We call the appropriate initializer after instantiation because
	// WithStartFunctions() skips auto-start to prevent wazero from
	// closing the module on proc_exit(0).
	if initFn := module.ExportedFunction("_initialize"); initFn != nil {
		if _, err := initFn.Call(ctx); err != nil {
			module.Close(ctx)
			return nil, fmt.Errorf("_initialize failed: %w", err)
		}
	}

	instance := &Instance{
		AgentID:        agentID,
		WASMBytes:      wasmBytes,
		Compiled:       compiled,
		Module:         module,
		Engine:         engine,
		Storage:        storageProvider,
		State:          nil,
		Budget:         budget,
		PricePerSecond: pricePerSecond,
		Manifest:       capManifest,
		EventLog:       el,
		TickNumber:     0,
		logger:         logger,
	}

	// Verify required exports exist
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
func (i *Instance) Tick(ctx context.Context) error {
	// Check budget before execution
	if i.Budget <= 0 {
		return fmt.Errorf("budget exhausted: %s", budget.Format(i.Budget))
	}

	// Capture pre-tick state for replay verification
	preState, err := i.captureState(ctx)
	if err != nil {
		return fmt.Errorf("pre-tick checkpoint failed: %w", err)
	}

	// Advance tick counter and begin event log recording
	i.TickNumber++
	i.EventLog.BeginTick(i.TickNumber)

	// Enforce tick timeout
	tickCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	fn := i.Module.ExportedFunction("agent_tick")
	if fn == nil {
		return fmt.Errorf("agent_tick function not found")
	}

	start := time.Now()
	_, tickErr := fn.Call(tickCtx)
	elapsed := time.Since(start)

	// Seal the event log regardless of tick success/failure
	sealed := i.EventLog.SealTick()

	if tickErr != nil {
		return fmt.Errorf("agent_tick failed: %w", tickErr)
	}

	// Capture post-tick state for replay verification
	postState, err := i.captureState(ctx)
	if err != nil {
		return fmt.Errorf("post-tick checkpoint failed: %w", err)
	}

	// Store replay verification data
	i.PreTickState = preState
	i.PostTickState = postState
	i.LastTickLog = sealed

	// Calculate and deduct execution cost (integer arithmetic, no float precision loss)
	costMicrocents := elapsed.Microseconds() * i.PricePerSecond / budget.MicrocentScale
	i.Budget -= costMicrocents

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
	)

	return nil
}

// captureState extracts the agent's current state via checkpoint exports.
func (i *Instance) captureState(ctx context.Context) ([]byte, error) {
	fnSize := i.Module.ExportedFunction("agent_checkpoint")
	sizeResults, err := fnSize.Call(ctx)
	if err != nil {
		return nil, fmt.Errorf("agent_checkpoint: %w", err)
	}
	size := uint32(sizeResults[0])
	if size == 0 {
		return []byte{}, nil
	}

	fnPtr := i.Module.ExportedFunction("agent_checkpoint_ptr")
	ptrResults, err := fnPtr.Call(ctx)
	if err != nil {
		return nil, fmt.Errorf("agent_checkpoint_ptr: %w", err)
	}
	ptr := uint32(ptrResults[0])

	data, ok := i.Module.Memory().Read(ptr, size)
	if !ok {
		return nil, fmt.Errorf("failed to read state from WASM memory")
	}

	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
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
// The checkpoint includes budget metadata and agent state.
func (i *Instance) SaveCheckpointToStorage(ctx context.Context) error {
	// Checkpoint agent state
	state, err := i.Checkpoint(ctx)
	if err != nil {
		return fmt.Errorf("failed to checkpoint agent: %w", err)
	}

	// Create checkpoint with budget metadata (v1 format)
	// Format: [version:1][budget:8][pricePerSecond:8][state:...]
	checkpoint := make([]byte, checkpointHeaderLen+len(state))
	checkpoint[0] = checkpointVersion
	binary.LittleEndian.PutUint64(checkpoint[1:9], uint64(i.Budget))
	binary.LittleEndian.PutUint64(checkpoint[9:17], uint64(i.PricePerSecond))
	copy(checkpoint[17:], state)

	// Save to storage provider
	if err := i.Storage.SaveCheckpoint(ctx, i.AgentID, checkpoint); err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
}

// LoadCheckpointFromStorage loads checkpoint from storage and resumes agent.
// The checkpoint includes budget metadata and agent state.
func (i *Instance) LoadCheckpointFromStorage(ctx context.Context) error {
	// Load from storage provider
	checkpoint, err := i.Storage.LoadCheckpoint(ctx, i.AgentID)
	if err != nil {
		if err == storage.ErrCheckpointNotFound {
			// No checkpoint exists - this is normal for new agents
			i.logger.Info("No existing checkpoint found", "agent_id", i.AgentID)
			return nil
		}
		return fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// Parse checkpoint v1 format: [version:1][budget:8][pricePerSecond:8][state:...]
	if len(checkpoint) < checkpointHeaderLen {
		return fmt.Errorf("invalid checkpoint format: too short")
	}
	if checkpoint[0] != checkpointVersion {
		return fmt.Errorf("unsupported checkpoint version: %d", checkpoint[0])
	}

	// Extract budget metadata (int64 microcents)
	restoredBudget := int64(binary.LittleEndian.Uint64(checkpoint[1:9]))
	restoredPrice := int64(binary.LittleEndian.Uint64(checkpoint[9:17]))
	state := checkpoint[17:]

	// Update instance budget
	i.Budget = restoredBudget
	i.PricePerSecond = restoredPrice

	i.logger.Info("Budget restored from checkpoint",
		"agent_id", i.AgentID,
		"budget", budget.Format(restoredBudget),
		"price_per_second", budget.Format(restoredPrice),
	)

	// Resume agent from state
	if err := i.Resume(ctx, state); err != nil {
		return fmt.Errorf("failed to resume agent: %w", err)
	}

	return nil
}

// ExtractAgentState extracts the agent state portion from a v1 checkpoint.
func ExtractAgentState(checkpoint []byte) ([]byte, error) {
	if len(checkpoint) < checkpointHeaderLen {
		return nil, fmt.Errorf("checkpoint too short: %d bytes", len(checkpoint))
	}
	if checkpoint[0] != checkpointVersion {
		return nil, fmt.Errorf("unsupported checkpoint version: %d", checkpoint[0])
	}
	return checkpoint[checkpointHeaderLen:], nil
}

// Close releases agent resources.
func (i *Instance) Close(ctx context.Context) error {
	if i.Module != nil {
		return i.Module.Close(ctx)
	}
	return nil
}
