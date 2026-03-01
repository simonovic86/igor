//go:build tinygo || wasip1

package main

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

// Igor hostcall imports — these are provided by the igor host module.
// Only available if the corresponding capability is declared in the manifest.

//go:wasmimport igor clock_now
func clockNow() int64

//go:wasmimport igor rand_bytes
func randBytes(ptr uint32, length uint32) int32

//go:wasmimport igor log_emit
func logEmit(ptr uint32, length uint32)

// logMsg sends a message through the igor log_emit hostcall.
func logMsg(msg string) {
	if len(msg) == 0 {
		return
	}
	buf := []byte(msg)
	logEmit(uint32(uintptr(unsafe.Pointer(&buf[0]))), uint32(len(buf)))
}

// State represents the agent's persistent state
type State struct {
	Counter uint64
}

var state State

// agent_init is called when the agent first starts
//
//export agent_init
func agent_init() {
	state.Counter = 0
	logMsg("initialized with counter=0")
}

// agent_tick is called periodically by the runtime
//
//export agent_tick
func agent_tick() {
	state.Counter++

	// Observe current time through capability membrane
	now := clockNow()

	// Get some random bytes to demonstrate rand capability
	var randBuf [4]byte
	randBytes(uint32(uintptr(unsafe.Pointer(&randBuf[0]))), 4)

	logMsg(fmt.Sprintf("tick %d time=%d rand=%x", state.Counter, now, randBuf))
}

// agent_checkpoint serializes the agent's state and returns size
//
//export agent_checkpoint
func agent_checkpoint() uint32 {
	logMsg(fmt.Sprintf("checkpoint counter=%d", state.Counter))
	return 8 // Size of uint64
}

// agent_checkpoint_ptr returns pointer to checkpoint data
//
//export agent_checkpoint_ptr
func agent_checkpoint_ptr() uint32 {
	return uint32(uintptr(unsafe.Pointer(&state.Counter)))
}

// agent_resume restores the agent from checkpointed state
//
//export agent_resume
func agent_resume(ptr, size uint32) {
	if size == 0 {
		logMsg("resume: empty state, keeping counter at 0")
		return
	}

	buf := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
	state.Counter = binary.LittleEndian.Uint64(buf)

	logMsg(fmt.Sprintf("resumed with counter=%d", state.Counter))
}

func main() {}
