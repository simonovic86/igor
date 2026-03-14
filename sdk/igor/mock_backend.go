// SPDX-License-Identifier: Apache-2.0

package igor

// MockBackend is the interface for mock hostcall implementations.
// Used by sdk/igor/mock to enable native (non-WASM) testing of agent logic.
type MockBackend interface {
	ClockNow() int64
	RandBytes(buf []byte) error
	LogEmit(msg string)
	WalletBalance() int64
	WalletReceiptCount() int
	WalletReceipt(index int) ([]byte, error)
	NodePrice() int64
	HTTPRequest(method, url string, headers map[string]string, body []byte) (statusCode int, respBody []byte, err error)
	WalletPay(amount int64, recipient, memo string) (receipt []byte, err error)
}

// activeMock is set by mock.Enable() and cleared by mock.Disable().
// Only referenced from hostcalls_wrappers_stub.go (non-WASM builds).
var activeMock MockBackend

// SetMockBackend installs or removes a mock hostcall backend.
// Pass nil to remove. Only effective in non-WASM builds.
func SetMockBackend(m MockBackend) {
	activeMock = m
}
