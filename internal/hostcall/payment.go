// SPDX-License-Identifier: Apache-2.0

package hostcall

import (
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"time"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/simonovic86/igor/pkg/manifest"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const (
	maxRecipientBytes = 256
	maxMemoBytes      = 512

	// Payment hostcall error codes (negative i32).
	payErrInsufficientBudget int32 = -1
	payErrInputTooLong       int32 = -2
	payErrRecipientBlocked   int32 = -3
	payErrAmountExceeded     int32 = -4
	payErrBufferTooSmall     int32 = -5
	payErrProcessing         int32 = -6
)

// WalletPayState provides budget deduction and signing for the wallet_pay hostcall.
type WalletPayState interface {
	GetBudget() int64
	DeductBudget(amount int64) error
	GetAgentPubKey() ed25519.PublicKey
	SignPayment(data []byte) []byte
}

// registerPayment registers the wallet_pay hostcall on the igor WASM host module.
//
// ABI:
//
//	wallet_pay(
//	  amount_microcents i64,
//	  recipient_ptr, recipient_len i32,
//	  memo_ptr, memo_len i32,
//	  receipt_ptr, receipt_cap i32
//	) -> i32
//
// Returns bytes written to receipt buffer on success, negative error code on failure.
// Receipt layout: [receipt_len: 4 bytes LE][receipt: N bytes].
func (r *Registry) registerPayment(builder wazero.HostModuleBuilder, state WalletPayState, capCfg manifest.CapabilityConfig) {
	allowedRecipients := extractAllowedRecipients(capCfg)
	maxPayment := int64(extractIntOption(capCfg.Options, "max_payment_microcents", 0))

	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module,
			amount int64,
			recipientPtr, recipientLen,
			memoPtr, memoLen,
			receiptPtr, receiptCap uint32,
		) int32 {
			// Validate input sizes.
			if recipientLen > maxRecipientBytes || memoLen > maxMemoBytes {
				return payErrInputTooLong
			}

			// Read recipient from WASM memory.
			recipientData, ok := m.Memory().Read(recipientPtr, recipientLen)
			if !ok {
				return payErrProcessing
			}
			recipient := string(recipientData)

			// Read memo from WASM memory.
			var memo string
			if memoLen > 0 {
				memoData, ok := m.Memory().Read(memoPtr, memoLen)
				if !ok {
					return payErrProcessing
				}
				memo = string(memoData)
			}

			// Validate amount.
			if amount <= 0 {
				return payErrProcessing
			}
			if maxPayment > 0 && amount > maxPayment {
				r.logger.Warn("Payment exceeds cap",
					"amount", amount,
					"max", maxPayment)
				return payErrAmountExceeded
			}

			// Validate recipient against allowed list.
			if len(allowedRecipients) > 0 {
				allowed := false
				for _, ar := range allowedRecipients {
					if recipient == ar {
						allowed = true
						break
					}
				}
				if !allowed {
					r.logger.Warn("Payment recipient blocked",
						"recipient", recipient)
					return payErrRecipientBlocked
				}
			}

			// Check sufficient budget.
			if state.GetBudget() < amount {
				r.logger.Warn("Insufficient budget for payment",
					"amount", amount,
					"budget", state.GetBudget())
				return payErrInsufficientBudget
			}

			// Deduct budget.
			if err := state.DeductBudget(amount); err != nil {
				r.logger.Error("Budget deduction failed", "error", err)
				return payErrProcessing
			}

			// Build signed payment receipt.
			// Format: [amount:8][timestamp:8][recipient_len:4][recipient][memo_len:4][memo][pubkey:32]
			pubKey := state.GetAgentPubKey()
			receiptDataLen := 8 + 8 + 4 + len(recipient) + 4 + len(memo) + len(pubKey)
			receiptData := make([]byte, receiptDataLen)
			off := 0
			binary.LittleEndian.PutUint64(receiptData[off:], uint64(amount))
			off += 8
			ts := time.Now().UnixNano()
			binary.LittleEndian.PutUint64(receiptData[off:], uint64(ts))
			off += 8
			binary.LittleEndian.PutUint32(receiptData[off:], uint32(len(recipient)))
			off += 4
			copy(receiptData[off:], recipient)
			off += len(recipient)
			binary.LittleEndian.PutUint32(receiptData[off:], uint32(len(memo)))
			off += 4
			copy(receiptData[off:], memo)
			off += len(memo)
			copy(receiptData[off:], pubKey)

			sig := state.SignPayment(receiptData)

			// Full receipt: [receiptData][signature]
			fullReceipt := make([]byte, len(receiptData)+len(sig))
			copy(fullReceipt, receiptData)
			copy(fullReceipt[len(receiptData):], sig)

			// Check receipt buffer capacity.
			needed := uint32(4 + len(fullReceipt))
			if needed > receiptCap {
				// Write size hint.
				if receiptCap >= 4 {
					sizeBuf := make([]byte, 4)
					binary.LittleEndian.PutUint32(sizeBuf, uint32(len(fullReceipt)))
					m.Memory().Write(receiptPtr, sizeBuf)
				}
				return payErrBufferTooSmall
			}

			// Write receipt: [receipt_len: 4 bytes LE][receipt: N bytes].
			out := make([]byte, needed)
			binary.LittleEndian.PutUint32(out[:4], uint32(len(fullReceipt)))
			copy(out[4:], fullReceipt)
			if !m.Memory().Write(receiptPtr, out) {
				return payErrProcessing
			}

			// Record observation for replay (CM-4).
			obsPayload := make([]byte, 8+len(fullReceipt))
			binary.LittleEndian.PutUint64(obsPayload[:8], uint64(amount))
			copy(obsPayload[8:], fullReceipt)
			r.eventLog.Record(eventlog.WalletPay, obsPayload)

			r.logger.Info("Payment processed",
				"amount", amount,
				"recipient", recipient,
				"memo", memo,
				"receipt_bytes", len(fullReceipt))

			return int32(needed)
		}).
		Export("wallet_pay")
}

// extractAllowedRecipients reads the allowed_recipients option from the capability config.
func extractAllowedRecipients(cfg manifest.CapabilityConfig) []string {
	if cfg.Options == nil {
		return nil
	}
	raw, ok := cfg.Options["allowed_recipients"]
	if !ok {
		return nil
	}
	slice, ok := raw.([]any)
	if !ok {
		return nil
	}
	recipients := make([]string, 0, len(slice))
	for _, v := range slice {
		if s, ok := v.(string); ok {
			recipients = append(recipients, s)
		}
	}
	return recipients
}
