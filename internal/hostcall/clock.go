// SPDX-License-Identifier: Apache-2.0

package hostcall

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/tetratelabs/wazero"
)

// registerClock adds clock_now to the host module builder.
// clock_now() -> i64: returns current time as Unix nanoseconds.
// Observation hostcall — recorded in event log for replay (CE-3).
func (r *Registry) registerClock(builder wazero.HostModuleBuilder) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int64 {
			now := time.Now().UnixNano()
			payload := binary.LittleEndian.AppendUint64(nil, uint64(now))
			r.eventLog.Record(eventlog.ClockNow, payload)
			return now
		}).
		Export("clock_now")
}
