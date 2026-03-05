// SPDX-License-Identifier: Apache-2.0

//go:build ignore

package main

import (
	"crypto/sha256"
	"encoding/json"
	"os"

	protomsg "github.com/simonovic86/igor/pkg/protocol"
)

func main() {
	wasmBinary := []byte{0x00, 0x61, 0x73, 0x6D} // WASM magic bytes
	wasmHash := sha256.Sum256(wasmBinary)

	// agent_package.json: without replay data
	pkg := protomsg.AgentPackage{
		AgentID:        "golden-agent",
		WASMBinary:     wasmBinary,
		WASMHash:       wasmHash[:],
		Checkpoint:     []byte{0x02, 0x01, 0x02, 0x03},
		ManifestData:   []byte("{}"),
		Budget:         1000000,
		PricePerSecond: 1000,
	}

	data, _ := json.MarshalIndent(pkg, "", "  ")
	os.WriteFile("agent_package.json", data, 0644)

	// agent_package_with_replay.json: with replay data
	pkgReplay := protomsg.AgentPackage{
		AgentID:        "golden-replay-agent",
		WASMBinary:     wasmBinary,
		WASMHash:       wasmHash[:],
		Checkpoint:     []byte{0x02, 0x01, 0x02, 0x03},
		ManifestData:   []byte(`{"capabilities":{"clock":{"version":1}}}`),
		Budget:         5000000,
		PricePerSecond: 2000,
		ReplayData: &protomsg.ReplayData{
			PreTickState: []byte{0xAA, 0xBB, 0xCC, 0xDD},
			TickNumber:   42,
			Entries: []protomsg.ReplayEntry{
				{HostcallID: 1, Payload: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}},
				{HostcallID: 2, Payload: []byte{0xDE, 0xAD, 0xBE, 0xEF}},
				{HostcallID: 3, Payload: []byte("tick")},
			},
		},
	}

	dataReplay, _ := json.MarshalIndent(pkgReplay, "", "  ")
	os.WriteFile("agent_package_with_replay.json", dataReplay, 0644)
}
