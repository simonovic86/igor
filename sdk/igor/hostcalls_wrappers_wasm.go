// SPDX-License-Identifier: Apache-2.0

// Package igor provides the Agent SDK for building Igor agents.
//
// It wraps the low-level WASM hostcall interface and lifecycle exports,
// letting agent authors focus on application logic rather than memory
// management, unsafe pointers, and serialization plumbing.
//
// Usage:
//
//	type MyAgent struct { Counter uint64 }
//	func (a *MyAgent) Init()                   { }
//	func (a *MyAgent) Tick()                   { a.Counter++; igor.Logf("tick %d", a.Counter) }
//	func (a *MyAgent) Marshal() []byte         { /* serialize */ }
//	func (a *MyAgent) Unmarshal(data []byte)   { /* deserialize */ }
//	func init() { igor.Run(&MyAgent{}) }

//go:build tinygo || wasip1

package igor

import (
	"fmt"
	"unsafe"
)

// ClockNow returns the current time as Unix nanoseconds.
// Requires the "clock" capability in the agent manifest.
func ClockNow() int64 {
	return clockNow()
}

// RandBytes fills buf with cryptographically random bytes.
// Requires the "rand" capability in the agent manifest.
// Returns an error if the hostcall fails.
func RandBytes(buf []byte) error {
	if len(buf) == 0 {
		return nil
	}
	rc := randBytes(
		uint32(uintptr(unsafe.Pointer(&buf[0]))),
		uint32(len(buf)),
	)
	if rc != 0 {
		return fmt.Errorf("rand_bytes failed: code %d", rc)
	}
	return nil
}

// Log emits a structured log message through the runtime.
// Requires the "log" capability in the agent manifest.
func Log(msg string) {
	if len(msg) == 0 {
		return
	}
	buf := []byte(msg)
	logEmit(
		uint32(uintptr(unsafe.Pointer(&buf[0]))),
		uint32(len(buf)),
	)
}

// Logf formats and emits a structured log message through the runtime.
// Requires the "log" capability in the agent manifest.
func Logf(format string, args ...any) {
	Log(fmt.Sprintf(format, args...))
}
