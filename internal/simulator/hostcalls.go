package simulator

import (
	"context"
	"encoding/binary"
	"log/slog"
	"math/rand/v2"
	"sync"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/simonovic86/igor/pkg/manifest"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// deterministicHostcalls registers hostcalls with controlled return values
// while recording observations to the event log (for replay verification).
type deterministicHostcalls struct {
	mu         sync.Mutex
	clockValue int64
	clockDelta int64
	randSrc    *rand.Rand
	eventLog   *eventlog.EventLog
	logger     *slog.Logger
	budget     int64
	nodePrice  int64
}

func newDeterministicHostcalls(
	clockStart, clockDelta int64,
	randSeed uint64,
	nodePrice int64,
	el *eventlog.EventLog,
	logger *slog.Logger,
) *deterministicHostcalls {
	return &deterministicHostcalls{
		clockValue: clockStart,
		clockDelta: clockDelta,
		randSrc:    rand.New(rand.NewPCG(randSeed, randSeed)),
		eventLog:   el,
		logger:     logger,
		nodePrice:  nodePrice,
	}
}

func (d *deterministicHostcalls) registerHostModule(
	ctx context.Context,
	rt wazero.Runtime,
	m *manifest.CapabilityManifest,
) error {
	builder := rt.NewHostModuleBuilder("igor")
	registered := 0

	if m.Has("clock") {
		d.registerClock(builder)
		registered++
	}

	if m.Has("rand") {
		d.registerRand(builder)
		registered++
	}

	if m.Has("log") {
		d.registerLog(builder)
		registered++
	}

	if m.Has("wallet") {
		d.registerWallet(builder)
		registered++
	}

	if m.Has("pricing") {
		d.registerPricing(builder)
		registered++
	}

	if registered == 0 {
		return nil
	}

	_, err := builder.Instantiate(ctx)
	return err
}

func (d *deterministicHostcalls) registerClock(builder wazero.HostModuleBuilder) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int64 {
			d.mu.Lock()
			now := d.clockValue
			d.clockValue += d.clockDelta
			d.mu.Unlock()

			payload := binary.LittleEndian.AppendUint64(nil, uint64(now))
			d.eventLog.Record(eventlog.ClockNow, payload)
			return now
		}).
		Export("clock_now")
}

func (d *deterministicHostcalls) registerRand(builder wazero.HostModuleBuilder) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, ptr, length uint32) int32 {
			d.mu.Lock()
			buf := make([]byte, length)
			for i := range buf {
				buf[i] = byte(d.randSrc.Uint32())
			}
			d.mu.Unlock()

			d.eventLog.Record(eventlog.RandBytes, buf)

			if !m.Memory().Write(ptr, buf) {
				return -4
			}
			return 0
		}).
		Export("rand_bytes")
}

func (d *deterministicHostcalls) registerLog(builder wazero.HostModuleBuilder) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, ptr, length uint32) {
			data, ok := m.Memory().Read(ptr, length)
			if !ok {
				return
			}

			d.eventLog.Record(eventlog.LogEmit, data)
			d.logger.Info("[agent]", "msg", string(data))
		}).
		Export("log_emit")
}

func (d *deterministicHostcalls) registerWallet(builder wazero.HostModuleBuilder) {
	// wallet_balance() -> i64
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int64 {
			d.mu.Lock()
			balance := d.budget
			d.mu.Unlock()
			payload := binary.LittleEndian.AppendUint64(nil, uint64(balance))
			d.eventLog.Record(eventlog.WalletBalance, payload)
			return balance
		}).
		Export("wallet_balance")

	// wallet_receipt_count() -> i32
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int32 {
			// Simulator has no receipts
			payload := binary.LittleEndian.AppendUint32(nil, 0)
			d.eventLog.Record(eventlog.WalletReceiptCount, payload)
			return 0
		}).
		Export("wallet_receipt_count")

	// wallet_receipt(index i32, buf_ptr i32, buf_len i32) -> i32
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, _ api.Module, _ int32, _, _ uint32) int32 {
			// Simulator has no receipts — always return error
			d.eventLog.Record(eventlog.WalletReceipt, nil)
			return -1
		}).
		Export("wallet_receipt")
}

func (d *deterministicHostcalls) registerPricing(builder wazero.HostModuleBuilder) {
	// node_price() -> i64
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int64 {
			d.mu.Lock()
			price := d.nodePrice
			d.mu.Unlock()
			payload := binary.LittleEndian.AppendUint64(nil, uint64(price))
			d.eventLog.Record(eventlog.NodePrice, payload)
			return price
		}).
		Export("node_price")
}
