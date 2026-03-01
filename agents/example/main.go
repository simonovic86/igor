//go:build tinygo || wasip1

package main

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

// Igor hostcall imports — provided by the igor host module.
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

// State layout (28 bytes, little-endian):
//
//	[0:8]   TickCount uint64  — total ticks ever executed
//	[8:16]  BirthNano int64   — clock_now on first tick (0 = unset)
//	[16:24] LastNano  int64   — clock_now on most recent tick
//	[24:28] Luck      uint32  — running XOR of random bytes
const stateSize = 28

var (
	tickCount uint64
	birthNano int64
	lastNano  int64
	luck      uint32

	// Serialization buffer for checkpoint
	ckptBuf [stateSize]byte
)

// serialize writes agent state to ckptBuf.
func serialize() {
	binary.LittleEndian.PutUint64(ckptBuf[0:8], tickCount)
	binary.LittleEndian.PutUint64(ckptBuf[8:16], uint64(birthNano))
	binary.LittleEndian.PutUint64(ckptBuf[16:24], uint64(lastNano))
	binary.LittleEndian.PutUint32(ckptBuf[24:28], luck)
}

// deserialize reads agent state from a byte slice.
func deserialize(buf []byte) {
	tickCount = binary.LittleEndian.Uint64(buf[0:8])
	birthNano = int64(binary.LittleEndian.Uint64(buf[8:16]))
	lastNano = int64(binary.LittleEndian.Uint64(buf[16:24]))
	luck = binary.LittleEndian.Uint32(buf[24:28])
}

//export agent_init
func agent_init() {
	tickCount = 0
	birthNano = 0
	lastNano = 0
	luck = 0
	// No hostcalls here — only agent_tick should call observation hostcalls
	// so that replay verification (CM-4) works correctly.
}

//export agent_tick
func agent_tick() {
	tickCount++

	// Observe current time
	now := clockNow()
	if birthNano == 0 {
		birthNano = now
	}
	lastNano = now

	// Get random bytes and accumulate into luck
	var randBuf [4]byte
	randBytes(uint32(uintptr(unsafe.Pointer(&randBuf[0]))), 4)
	luck ^= binary.LittleEndian.Uint32(randBuf[:])

	// Calculate uptime in seconds
	ageSec := (lastNano - birthNano) / 1_000_000_000

	// Log narrative
	logMsg(fmt.Sprintf("[survivor] tick %d | age %ds | luck 0x%08x",
		tickCount, ageSec, luck))

	// Milestones every 10 ticks
	if tickCount%10 == 0 {
		logMsg(fmt.Sprintf("[survivor] milestone: survived %d ticks", tickCount))
	}
}

//export agent_checkpoint
func agent_checkpoint() uint32 {
	serialize()
	return stateSize
}

//export agent_checkpoint_ptr
func agent_checkpoint_ptr() uint32 {
	return uint32(uintptr(unsafe.Pointer(&ckptBuf[0])))
}

//export agent_resume
func agent_resume(ptr, size uint32) {
	if size == 0 {
		return
	}
	// Pure state restoration — no side effects.
	// agent_resume is called during replay verification (CM-4),
	// so it must produce identical state to what was checkpointed.
	buf := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
	deserialize(buf)
}

func main() {}
