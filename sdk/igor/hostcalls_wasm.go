// SPDX-License-Identifier: Apache-2.0

//go:build tinygo || wasip1

package igor

// Raw WASM imports from the igor host module.
// These are the low-level hostcall declarations; use the safe wrappers
// in hostcalls.go instead.

//go:wasmimport igor clock_now
func clockNow() int64

//go:wasmimport igor rand_bytes
func randBytes(ptr uint32, length uint32) int32

//go:wasmimport igor log_emit
func logEmit(ptr uint32, length uint32)
