// SPDX-License-Identifier: Apache-2.0

package protocol

import (
	"encoding/json"
	"testing"
)

func TestAgentPackage_JSONRoundTrip(t *testing.T) {
	pkg := AgentPackage{
		AgentID:        "agent-1",
		WASMBinary:     []byte{0x00, 0x61, 0x73, 0x6D},
		WASMHash:       make([]byte, 32),
		Checkpoint:     []byte("checkpoint-data"),
		Budget:         1000000,
		PricePerSecond: 1000,
	}
	data, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded AgentPackage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.AgentID != pkg.AgentID {
		t.Errorf("AgentID: got %q, want %q", decoded.AgentID, pkg.AgentID)
	}
	if decoded.Budget != pkg.Budget {
		t.Errorf("Budget: got %d, want %d", decoded.Budget, pkg.Budget)
	}
	if decoded.PricePerSecond != pkg.PricePerSecond {
		t.Errorf("PricePerSecond: got %d, want %d", decoded.PricePerSecond, pkg.PricePerSecond)
	}
	if len(decoded.WASMBinary) != len(pkg.WASMBinary) {
		t.Errorf("WASMBinary length: got %d, want %d", len(decoded.WASMBinary), len(pkg.WASMBinary))
	}
}

func TestAgentTransfer_JSONRoundTrip(t *testing.T) {
	transfer := AgentTransfer{
		Package:      AgentPackage{AgentID: "test-agent", Budget: 500000},
		SourceNodeID: "node-1",
	}
	data, err := json.Marshal(transfer)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded AgentTransfer
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.SourceNodeID != "node-1" {
		t.Errorf("SourceNodeID: got %q, want %q", decoded.SourceNodeID, "node-1")
	}
	if decoded.Package.AgentID != "test-agent" {
		t.Errorf("Package.AgentID: got %q, want %q", decoded.Package.AgentID, "test-agent")
	}
}

func TestAgentStarted_JSONRoundTrip(t *testing.T) {
	started := AgentStarted{
		AgentID: "agent-1",
		NodeID:  "node-2",
		Success: true,
	}
	data, err := json.Marshal(started)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded AgentStarted
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !decoded.Success {
		t.Error("expected Success=true")
	}
	if decoded.AgentID != "agent-1" {
		t.Errorf("AgentID: got %q, want %q", decoded.AgentID, "agent-1")
	}
	if decoded.NodeID != "node-2" {
		t.Errorf("NodeID: got %q, want %q", decoded.NodeID, "node-2")
	}
}

func TestAgentStarted_ErrorRoundTrip(t *testing.T) {
	started := AgentStarted{
		AgentID: "agent-fail",
		NodeID:  "node-3",
		Success: false,
		Error:   "capability check failed",
	}
	data, err := json.Marshal(started)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded AgentStarted
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.Success {
		t.Error("expected Success=false")
	}
	if decoded.Error != "capability check failed" {
		t.Errorf("Error: got %q, want %q", decoded.Error, "capability check failed")
	}
}

func TestAgentPackage_ReplayDataOmitted(t *testing.T) {
	pkg := AgentPackage{
		AgentID:    "agent-no-replay",
		ReplayData: nil,
	}
	data, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// ReplayData should be omitted from JSON when nil
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal raw: %v", err)
	}
	if _, ok := raw["ReplayData"]; ok {
		t.Error("ReplayData should be omitted when nil")
	}
}
