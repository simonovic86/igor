package hostcall

import (
	"context"
	crypto_rand "crypto/rand"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// registerRand adds rand_bytes to the host module builder.
// rand_bytes(ptr i32, len i32) -> i32: fills buffer with random bytes.
// Returns 0 on success, -1 on internal error, -4 if buffer write fails.
// Observation hostcall — recorded in event log for replay (CE-3).
func (r *Registry) registerRand(builder wazero.HostModuleBuilder) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, ptr, length uint32) int32 {
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
