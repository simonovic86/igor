//go:build tinygo || wasip1

package igor

import (
	"fmt"
	"unsafe"
)

// Raw WASM imports for wallet hostcalls from the igor host module.

//go:wasmimport igor wallet_balance
func walletBalance() int64

//go:wasmimport igor wallet_receipt_count
func walletReceiptCount() int32

//go:wasmimport igor wallet_receipt
func walletReceipt(index int32, ptr uint32, length uint32) int32

// WalletBalance returns the agent's current budget in microcents.
// Requires the "wallet" capability in the agent manifest.
func WalletBalance() int64 {
	return walletBalance()
}

// WalletReceiptCount returns the number of payment receipts available.
// Requires the "wallet" capability in the agent manifest.
func WalletReceiptCount() int {
	return int(walletReceiptCount())
}

// WalletReceipt reads the receipt at the given index into a byte slice.
// Requires the "wallet" capability in the agent manifest.
func WalletReceipt(index int) ([]byte, error) {
	buf := make([]byte, 1024)
	rc := walletReceipt(int32(index),
		uint32(uintptr(unsafe.Pointer(&buf[0]))),
		uint32(len(buf)),
	)
	if rc < 0 {
		return nil, fmt.Errorf("wallet_receipt failed: code %d", rc)
	}
	return buf[:rc], nil
}
