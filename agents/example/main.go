package main

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

// State represents the agent's persistent state
type State struct {
	Counter uint64
}

//nolint:unused // State used by WASM exports
var state State

// agent_init is called when the agent first starts
//
//export agent_init
//nolint:unused // Called by WASM runtime
func agent_init() {
	state.Counter = 0
	fmt.Println("[agent] Initialized with counter = 0")
}

// agent_tick is called periodically by the runtime
//
//export agent_tick
//nolint:unused // Called by WASM runtime
func agent_tick() {
	state.Counter++
	fmt.Printf("[agent] Tick %d\n", state.Counter)
}

// agent_checkpoint serializes the agent's state and returns size
// The state is written to memory starting at address returned by agent_checkpoint_ptr
//
//export agent_checkpoint
//nolint:unused // Called by WASM runtime
func agent_checkpoint() uint32 {
	fmt.Printf("[agent] Checkpoint: counter=%d\n", state.Counter)
	return 8 // Size of uint64
}

// agent_checkpoint_ptr returns pointer to checkpoint data
//
//export agent_checkpoint_ptr
//nolint:unused // Called by WASM runtime
func agent_checkpoint_ptr() uint32 {
	// Return pointer to the counter field directly
	return uint32(uintptr(unsafe.Pointer(&state.Counter)))
}

// agent_resume restores the agent from checkpointed state
//
//export agent_resume
//nolint:unused // Called by WASM runtime
func agent_resume(ptr, size uint32) {
	if size == 0 {
		fmt.Println("[agent] Resume: empty state, keeping counter at 0")
		return
	}

	// Deserialize state from memory
	//nolint:gosec // Unsafe pointer required for WASM memory access
	buf := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
	state.Counter = binary.LittleEndian.Uint64(buf)

	fmt.Printf("[agent] Resumed with counter=%d\n", state.Counter)
}

func main() {}
