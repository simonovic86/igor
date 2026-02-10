package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/simonovic86/igor/internal/runtime"
	"github.com/simonovic86/igor/internal/storage"
)

// Instance represents a running agent instance.
type Instance struct {
	AgentID  string
	Compiled wazero.CompiledModule
	Module   api.Module
	Engine   *runtime.Engine
	Storage  storage.Provider
	State    []byte
	logger   *slog.Logger
}

// LoadAgent loads and compiles a WASM agent from a file.
func LoadAgent(
	ctx context.Context,
	engine *runtime.Engine,
	wasmPath string,
	agentID string,
	storageProvider storage.Provider,
	logger *slog.Logger,
) (*Instance, error) {
	logger.Info("Loading agent", "agent_id", agentID, "path", wasmPath)

	// Compile WASM module
	compiled, err := engine.LoadWASM(ctx, wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load WASM: %w", err)
	}

	// Instantiate module
	module, err := engine.InstantiateModule(ctx, compiled, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate module: %w", err)
	}

	instance := &Instance{
		AgentID:  agentID,
		Compiled: compiled,
		Module:   module,
		Engine:   engine,
		Storage:  storageProvider,
		State:    nil,
		logger:   logger,
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

// Tick executes one tick of the agent with a timeout.
func (i *Instance) Tick(ctx context.Context) error {
	// Enforce tick timeout
	tickCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	fn := i.Module.ExportedFunction("agent_tick")
	if fn == nil {
		return fmt.Errorf("agent_tick function not found")
	}

	start := time.Now()
	_, err := fn.Call(tickCtx)
	elapsed := time.Since(start)

	if err != nil {
		return fmt.Errorf("agent_tick failed: %w", err)
	}

	i.logger.Debug("Tick completed",
		"agent_id", i.AgentID,
		"duration_ms", elapsed.Milliseconds(),
	)

	return nil
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
func (i *Instance) SaveCheckpointToStorage(ctx context.Context) error {
	// Checkpoint agent state
	state, err := i.Checkpoint(ctx)
	if err != nil {
		return fmt.Errorf("failed to checkpoint agent: %w", err)
	}

	// Save to storage provider
	if err := i.Storage.SaveCheckpoint(ctx, i.AgentID, state); err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
}

// LoadCheckpointFromStorage loads checkpoint from storage and resumes agent.
func (i *Instance) LoadCheckpointFromStorage(ctx context.Context) error {
	// Load from storage provider
	state, err := i.Storage.LoadCheckpoint(ctx, i.AgentID)
	if err != nil {
		if err == storage.ErrCheckpointNotFound {
			// No checkpoint exists - this is normal for new agents
			i.logger.Info("No existing checkpoint found", "agent_id", i.AgentID)
			return nil
		}
		return fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// Resume agent from state
	if err := i.Resume(ctx, state); err != nil {
		return fmt.Errorf("failed to resume agent: %w", err)
	}

	return nil
}

// Close releases agent resources.
func (i *Instance) Close(ctx context.Context) error {
	if i.Module != nil {
		return i.Module.Close(ctx)
	}
	return nil
}
