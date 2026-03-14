// SPDX-License-Identifier: Apache-2.0

//go:build !tinygo && !wasip1

package igor

// WalletPay sends a payment from the agent's budget to the given recipient.
// In non-WASM builds, dispatches to the registered MockBackend.
func WalletPay(amount int64, recipient, memo string) ([]byte, error) {
	if activeMock != nil {
		return activeMock.WalletPay(amount, recipient, memo)
	}
	panic("igor: WalletPay requires WASM runtime or mock (see sdk/igor/mock)")
}
