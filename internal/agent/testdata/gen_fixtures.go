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

	// checkpoint_v3.bin: 81-byte v0x03 header + 8-byte counter state
	// budget=2000000, price=1500, tick=10, epoch=(3,7), leaseExpiry=1700000000000000000, state=counter(5)
	v3 := make([]byte, 81+8)
	v3[0] = 0x03
	binary.LittleEndian.PutUint64(v3[1:9], 2000000)
	binary.LittleEndian.PutUint64(v3[9:17], 1500)
	binary.LittleEndian.PutUint64(v3[17:25], 10)
	copy(v3[25:57], hash[:])
	binary.LittleEndian.PutUint64(v3[57:65], 3)                   // majorVersion
	binary.LittleEndian.PutUint64(v3[65:73], 7)                   // leaseGeneration
	binary.LittleEndian.PutUint64(v3[73:81], 1700000000000000000) // leaseExpiry (unix nanos)
	binary.LittleEndian.PutUint64(v3[81:89], 5)                   // counter=5
	os.WriteFile("checkpoint_v3.bin", v3, 0644)
}
