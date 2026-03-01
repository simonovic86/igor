package hostcall

import (
	"context"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// registerLog adds log_emit to the host module builder.
// log_emit(ptr i32, len i32): emits a structured log entry.
// Not a side effect — logging is an observation that produces no
// externally visible state change.
func (r *Registry) registerLog(builder wazero.HostModuleBuilder) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, ptr, length uint32) {
			data, ok := m.Memory().Read(ptr, length)
			if !ok {
				return
			}

			r.eventLog.Record(eventlog.LogEmit, data)
			r.logger.Info("[agent]", "msg", string(data))
		}).
		Export("log_emit")
}
