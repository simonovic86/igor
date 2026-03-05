// Package wasmutil provides shared WASM module interaction helpers used by
// the agent, replay, and simulator packages. Extracting these avoids three
// near-identical copies of captureState and resumeAgent.
package wasmutil

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero/api"
)

// CaptureState extracts the agent's current state via the agent_checkpoint and
// agent_checkpoint_ptr exports. Returns a copy of the state bytes.
func CaptureState(ctx context.Context, mod api.Module) ([]byte, error) {
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
		return nil, fmt.Errorf("failed to read state from WASM memory")
	}

	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

// ResumeAgent restores agent state by calling malloc + agent_resume.
// Handles the empty-state case (ptr=0, len=0) without requiring malloc.
func ResumeAgent(ctx context.Context, mod api.Module, state []byte) error {
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
