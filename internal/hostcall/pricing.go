package hostcall

import (
	"context"
	"encoding/binary"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/tetratelabs/wazero"
)

// PricingState provides pricing hostcalls with access to node price configuration.
type PricingState interface {
	GetNodePrice() int64 // price per second in microcents
}

// registerPricing adds node_price to the host module builder.
// node_price is an observation hostcall recorded in the event log (CM-4).
func (r *Registry) registerPricing(builder wazero.HostModuleBuilder, ps PricingState) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int64 {
			price := ps.GetNodePrice()
			payload := binary.LittleEndian.AppendUint64(nil, uint64(price))
			r.eventLog.Record(eventlog.NodePrice, payload)
			return price
		}).
		Export("node_price")
}
