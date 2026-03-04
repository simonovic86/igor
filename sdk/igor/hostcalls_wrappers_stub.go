//go:build !tinygo && !wasip1

package igor

import "fmt"

// ClockNow returns the current time as Unix nanoseconds.
// In non-WASM builds, dispatches to the registered MockBackend.
func ClockNow() int64 {
	if activeMock != nil {
		return activeMock.ClockNow()
	}
	panic("igor: ClockNow requires WASM runtime or mock (see sdk/igor/mock)")
}

// RandBytes fills buf with random bytes.
// In non-WASM builds, dispatches to the registered MockBackend.
func RandBytes(buf []byte) error {
	if len(buf) == 0 {
		return nil
	}
	if activeMock != nil {
		return activeMock.RandBytes(buf)
	}
	panic("igor: RandBytes requires WASM runtime or mock (see sdk/igor/mock)")
}

// Log emits a structured log message.
// In non-WASM builds, dispatches to the registered MockBackend.
func Log(msg string) {
	if len(msg) == 0 {
		return
	}
	if activeMock != nil {
		activeMock.LogEmit(msg)
		return
	}
	panic("igor: Log requires WASM runtime or mock (see sdk/igor/mock)")
}

// Logf formats and emits a structured log message.
// In non-WASM builds, dispatches to the registered MockBackend.
func Logf(format string, args ...any) {
	Log(fmt.Sprintf(format, args...))
}

// WalletBalance returns the agent's current budget in microcents.
// In non-WASM builds, dispatches to the registered MockBackend.
func WalletBalance() int64 {
	if activeMock != nil {
		return activeMock.WalletBalance()
	}
	panic("igor: WalletBalance requires WASM runtime or mock (see sdk/igor/mock)")
}

// WalletReceiptCount returns the number of payment receipts available.
// In non-WASM builds, dispatches to the registered MockBackend.
func WalletReceiptCount() int {
	if activeMock != nil {
		return activeMock.WalletReceiptCount()
	}
	panic("igor: WalletReceiptCount requires WASM runtime or mock (see sdk/igor/mock)")
}

// WalletReceipt reads the receipt at the given index.
// In non-WASM builds, dispatches to the registered MockBackend.
func WalletReceipt(index int) ([]byte, error) {
	if activeMock != nil {
		return activeMock.WalletReceipt(index)
	}
	panic("igor: WalletReceipt requires WASM runtime or mock (see sdk/igor/mock)")
}
