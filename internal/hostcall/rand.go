// SPDX-License-Identifier: Apache-2.0

package hostcall

import (
	"context"
	crypto_rand "crypto/rand"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// maxRandBytes caps the buffer size for rand_bytes to prevent OOM panics.
// Matches the WASM linear memory limit (64 MB = 1024 pages × 64 KiB).
const maxRandBytes = 64 * 1024 * 1024

// registerRand adds rand_bytes to the host module builder.
// rand_bytes(ptr i32, len i32) -> i32: fills buffer with random bytes.
// Returns 0 on success, -1 on internal error, -2 if length exceeds maximum,
// -4 if buffer write fails.
// Observation hostcall — recorded in event log for replay (CE-3).
func (r *Registry) registerRand(builder wazero.HostModuleBuilder) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, ptr, length uint32) int32 {
			if length == 0 {
				return 0
			}
			if length > maxRandBytes {
				r.logger.Warn("rand_bytes: length exceeds maximum",
					"requested", length,
					"max", maxRandBytes,
				)
				return -2
			}
			buf := make([]byte, length)
			if _, err := crypto_rand.Read(buf); err != nil {
				r.logger.Error("rand_bytes: crypto/rand.Read failed", "error", err)
				return -1
			}

			r.eventLog.Record(eventlog.RandBytes, buf)

			if !m.Memory().Write(ptr, buf) {
				return -4 // buffer too small / out of bounds
			}
			return 0
		}).
		Export("rand_bytes")
}
