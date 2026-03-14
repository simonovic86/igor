// SPDX-License-Identifier: Apache-2.0

//go:build tinygo || wasip1

package igor

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

// Raw WASM import for the payment hostcall from the igor host module.

//go:wasmimport igor wallet_pay
func walletPay(
	amount int64,
	recipientPtr, recipientLen,
	memoPtr, memoLen,
	receiptPtr, receiptCap uint32,
) int32

// WalletPay sends a payment from the agent's budget to the given recipient.
// Requires the "x402" capability in the agent manifest.
// Returns the signed payment receipt bytes on success.
func WalletPay(amount int64, recipient, memo string) ([]byte, error) {
	recipientBuf := []byte(recipient)
	memoBuf := []byte(memo)

	// Initial receipt buffer: 1KB.
	receiptBuf := make([]byte, 1024)

	var recipientPtr, memoPtr, memoLen uint32
	recipientPtr = uint32(uintptr(unsafe.Pointer(&recipientBuf[0])))
	if len(memoBuf) > 0 {
		memoPtr = uint32(uintptr(unsafe.Pointer(&memoBuf[0])))
		memoLen = uint32(len(memoBuf))
	}
	receiptPtr := uint32(uintptr(unsafe.Pointer(&receiptBuf[0])))

	rc := walletPay(
		amount,
		recipientPtr, uint32(len(recipientBuf)),
		memoPtr, memoLen,
		receiptPtr, uint32(len(receiptBuf)),
	)

	// If receipt buffer too small, retry with the size hint.
	if rc == -5 && len(receiptBuf) >= 4 {
		needed := binary.LittleEndian.Uint32(receiptBuf[:4])
		if needed > 0 && needed <= 64*1024 { // Cap retry at 64KB.
			receiptBuf = make([]byte, needed+4) // +4 for the length prefix.
			receiptPtr = uint32(uintptr(unsafe.Pointer(&receiptBuf[0])))
			rc = walletPay(
				amount,
				uint32(uintptr(unsafe.Pointer(&recipientBuf[0]))), uint32(len(recipientBuf)),
				memoPtr, memoLen,
				receiptPtr, uint32(len(receiptBuf)),
			)
		}
	}

	if rc < 0 {
		switch rc {
		case -1:
			return nil, fmt.Errorf("wallet_pay: insufficient budget")
		case -2:
			return nil, fmt.Errorf("wallet_pay: input too long")
		case -3:
			return nil, fmt.Errorf("wallet_pay: recipient not allowed")
		case -4:
			return nil, fmt.Errorf("wallet_pay: amount exceeds cap")
		default:
			return nil, fmt.Errorf("wallet_pay failed: code %d", rc)
		}
	}

	// Parse receipt: [receipt_len: 4 bytes LE][receipt: N bytes].
	if len(receiptBuf) < 4 {
		return nil, nil
	}
	receiptLen := binary.LittleEndian.Uint32(receiptBuf[:4])
	if int(receiptLen)+4 > len(receiptBuf) {
		return nil, fmt.Errorf("wallet_pay: receipt truncated")
	}
	return receiptBuf[4 : 4+receiptLen], nil
}
