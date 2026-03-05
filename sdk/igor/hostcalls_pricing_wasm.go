//go:build tinygo || wasip1

package igor

// Raw WASM import for pricing hostcall from the igor host module.

//go:wasmimport igor node_price
func nodePrice() int64

// NodePrice returns the current node's execution price in microcents per second.
// Requires the "pricing" capability in the agent manifest.
func NodePrice() int64 {
	return nodePrice()
}
