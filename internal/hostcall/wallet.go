// SPDX-License-Identifier: Apache-2.0

package hostcall

import (
	"context"
	"encoding/binary"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// WalletState provides wallet hostcalls with access to agent budget and receipts.
// Decouples hostcalls from the agent.Instance type.
type WalletState interface {
	GetBudget() int64
	GetReceiptCount() int
	GetReceiptBytes(index int) ([]byte, error)
}

// registerWallet adds wallet_balance, wallet_receipt_count, and wallet_receipt
// to the host module builder. All are observation hostcalls recorded in the event log.
func (r *Registry) registerWallet(builder wazero.HostModuleBuilder, ws WalletState) {
	// wallet_balance() -> i64
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int64 {
			balance := ws.GetBudget()
			payload := binary.LittleEndian.AppendUint64(nil, uint64(balance))
			r.eventLog.Record(eventlog.WalletBalance, payload)
			return balance
		}).
		Export("wallet_balance")

	// wallet_receipt_count() -> i32
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) int32 {
			count := int32(ws.GetReceiptCount())
			payload := binary.LittleEndian.AppendUint32(nil, uint32(count))
			r.eventLog.Record(eventlog.WalletReceiptCount, payload)
			return count
		}).
		Export("wallet_receipt_count")

	// wallet_receipt(index i32, buf_ptr i32, buf_len i32) -> i32
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, index int32, ptr, length uint32) int32 {
			data, err := ws.GetReceiptBytes(int(index))
			if err != nil {
				r.logger.Warn("wallet_receipt: invalid index", "index", index, "error", err)
				return -1
			}
			if uint32(len(data)) > length {
				return -4 // buffer too small
			}
			r.eventLog.Record(eventlog.WalletReceipt, data)
			if !m.Memory().Write(ptr, data) {
				return -4
			}
			return int32(len(data))
		}).
		Export("wallet_receipt")
}
