//go:build ignore

package main

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
)

func main() {
	// Known WASM bytes for hash computation
	wasmBytes := []byte("known-wasm-binary-for-golden-test")
	hash := sha256.Sum256(wasmBytes)

	// checkpoint.bin: 57-byte header + 8-byte counter state
	// budget=1000000, price=1000, tick=5, wasmHash=sha256("known-wasm-binary-for-golden-test"), state=counter(3)
	ckpt := make([]byte, 57+8)
	ckpt[0] = 0x02
	binary.LittleEndian.PutUint64(ckpt[1:9], 1000000)
	binary.LittleEndian.PutUint64(ckpt[9:17], 1000)
	binary.LittleEndian.PutUint64(ckpt[17:25], 5)
	copy(ckpt[25:57], hash[:])
	binary.LittleEndian.PutUint64(ckpt[57:65], 3) // counter=3
	os.WriteFile("checkpoint.bin", ckpt, 0644)

	// checkpoint_empty_state.bin: 57-byte header, no state
	empty := make([]byte, 57)
	empty[0] = 0x02
	binary.LittleEndian.PutUint64(empty[1:9], 500000)
	binary.LittleEndian.PutUint64(empty[9:17], 2000)
	binary.LittleEndian.PutUint64(empty[17:25], 0)
	copy(empty[25:57], hash[:])
	os.WriteFile("checkpoint_empty_state.bin", empty, 0644)
}
