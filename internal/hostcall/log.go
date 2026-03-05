// SPDX-License-Identifier: Apache-2.0

package hostcall

import (
	"context"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// maxLogBytes caps the buffer size for log_emit to prevent excessive memory allocation.
const maxLogBytes = 1 * 1024 * 1024 // 1 MB

// registerLog adds log_emit to the host module builder.
// log_emit(ptr i32, len i32): emits a structured log entry.
// Not a side effect — logging is an observation that produces no
// externally visible state change.
func (r *Registry) registerLog(builder wazero.HostModuleBuilder) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, ptr, length uint32) {
			if length == 0 {
				return
			}
			if length > maxLogBytes {
				r.logger.Warn("log_emit: message too large, dropped",
					"length", length, "max", maxLogBytes)
				return
			}
			data, ok := m.Memory().Read(ptr, length)
			if !ok {
				r.logger.Warn("log_emit: failed to read from WASM memory",
					"ptr", ptr, "length", length)
				return
			}

			r.eventLog.Record(eventlog.LogEmit, data)
			r.logger.Info("[agent]", "msg", string(data))
		}).
		Export("log_emit")
}
