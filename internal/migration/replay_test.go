package migration

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/internal/eventlog"
	protomsg "github.com/simonovic86/igor/pkg/protocol"
)

func TestReplayDataFromInstance_EmptyWindow(t *testing.T) {
	inst := &agent.Instance{}
	rd := replayDataFromInstance(inst, nil)
	if rd != nil {
		t.Error("expected nil when ReplayWindow is empty")
	}
}

func TestReplayDataFromInstance_WithData(t *testing.T) {
	postState := []byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	// Build a v1 checkpoint that contains postState as the agent state
	checkpoint := make([]byte, 17+len(postState))
	checkpoint[0] = 0x01
	binary.LittleEndian.PutUint64(checkpoint[1:9], 1000000)
	binary.LittleEndian.PutUint64(checkpoint[9:17], 1000)
	copy(checkpoint[17:], postState)

	inst := &agent.Instance{
		ReplayWindow: []agent.TickSnapshot{
			{
				TickNumber: 5,
				PreState:   []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				PostState:  postState,
				TickLog: &eventlog.TickLog{
					TickNumber: 5,
					Entries: []eventlog.Entry{
						{HostcallID: eventlog.ClockNow, Payload: []byte{0xAA, 0xBB}},
						{HostcallID: eventlog.RandBytes, Payload: []byte{0xCC, 0xDD, 0xEE}},
						{HostcallID: eventlog.LogEmit, Payload: []byte("tick")},
					},
				},
			},
		},
	}

	rd := replayDataFromInstance(inst, checkpoint)
	if rd == nil {
		t.Fatal("expected non-nil ReplayData")
	}

	if rd.TickNumber != 5 {
		t.Errorf("TickNumber: got %d, want 5", rd.TickNumber)
	}
	if len(rd.PreTickState) != 8 {
		t.Errorf("PreTickState length: got %d, want 8", len(rd.PreTickState))
	}
	if len(rd.Entries) != 3 {
		t.Fatalf("Entries length: got %d, want 3", len(rd.Entries))
	}

	// Verify entry conversion
	if rd.Entries[0].HostcallID != uint16(eventlog.ClockNow) {
		t.Errorf("Entry[0] HostcallID: got %d, want %d", rd.Entries[0].HostcallID, eventlog.ClockNow)
	}
	if rd.Entries[1].HostcallID != uint16(eventlog.RandBytes) {
		t.Errorf("Entry[1] HostcallID: got %d, want %d", rd.Entries[1].HostcallID, eventlog.RandBytes)
	}
	if rd.Entries[2].HostcallID != uint16(eventlog.LogEmit) {
		t.Errorf("Entry[2] HostcallID: got %d, want %d", rd.Entries[2].HostcallID, eventlog.LogEmit)
	}
}

func TestReplayDataFromInstance_StalenessGuard(t *testing.T) {
	postState := []byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	differentState := []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	// Build checkpoint with differentState — does NOT match PostState
	checkpoint := make([]byte, 17+len(differentState))
	checkpoint[0] = 0x01
	binary.LittleEndian.PutUint64(checkpoint[1:9], 1000000)
	binary.LittleEndian.PutUint64(checkpoint[9:17], 1000)
	copy(checkpoint[17:], differentState)

	inst := &agent.Instance{
		ReplayWindow: []agent.TickSnapshot{
			{
				TickNumber: 5,
				PreState:   []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				PostState:  postState,
				TickLog: &eventlog.TickLog{
					TickNumber: 5,
					Entries:    []eventlog.Entry{{HostcallID: eventlog.ClockNow, Payload: []byte{0xAA}}},
				},
			},
		},
	}

	rd := replayDataFromInstance(inst, checkpoint)
	if rd != nil {
		t.Error("expected nil when PostState does not match checkpoint state (staleness guard)")
	}
}

func TestToTickLog_Conversion(t *testing.T) {
	rd := &protomsg.ReplayData{
		PreTickState: []byte{0x01},
		TickNumber:   42,
		Entries: []protomsg.ReplayEntry{
			{HostcallID: 1, Payload: []byte{0xAA}},
			{HostcallID: 2, Payload: []byte{0xBB, 0xCC}},
			{HostcallID: 3, Payload: []byte("msg")},
		},
	}

	tickLog := toTickLog(rd)
	if tickLog.TickNumber != 42 {
		t.Errorf("TickNumber: got %d, want 42", tickLog.TickNumber)
	}
	if len(tickLog.Entries) != 3 {
		t.Fatalf("Entries length: got %d, want 3", len(tickLog.Entries))
	}

	if tickLog.Entries[0].HostcallID != eventlog.ClockNow {
		t.Errorf("Entry[0] HostcallID: got %d, want %d", tickLog.Entries[0].HostcallID, eventlog.ClockNow)
	}
	if tickLog.Entries[1].HostcallID != eventlog.RandBytes {
		t.Errorf("Entry[1] HostcallID: got %d, want %d", tickLog.Entries[1].HostcallID, eventlog.RandBytes)
	}
	if tickLog.Entries[2].HostcallID != eventlog.LogEmit {
		t.Errorf("Entry[2] HostcallID: got %d, want %d", tickLog.Entries[2].HostcallID, eventlog.LogEmit)
	}
}

func TestReplayData_JSONRoundTrip(t *testing.T) {
	// Package with replay data
	original := protomsg.AgentPackage{
		AgentID:        "test-agent",
		WASMBinary:     []byte{0x00, 0x61, 0x73, 0x6D},
		Checkpoint:     []byte{0x01, 0x02, 0x03},
		ManifestData:   []byte("{}"),
		Budget:         1000000,
		PricePerSecond: 1000,
		ReplayData: &protomsg.ReplayData{
			PreTickState: []byte{0xAA, 0xBB},
			TickNumber:   7,
			Entries: []protomsg.ReplayEntry{
				{HostcallID: 1, Payload: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}},
				{HostcallID: 2, Payload: []byte{0xDE, 0xAD}},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded protomsg.AgentPackage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.ReplayData == nil {
		t.Fatal("expected non-nil ReplayData after round-trip")
	}
	if decoded.ReplayData.TickNumber != 7 {
		t.Errorf("TickNumber: got %d, want 7", decoded.ReplayData.TickNumber)
	}
	if len(decoded.ReplayData.Entries) != 2 {
		t.Errorf("Entries: got %d, want 2", len(decoded.ReplayData.Entries))
	}
	if decoded.ReplayData.Entries[0].HostcallID != 1 {
		t.Errorf("Entry[0] HostcallID: got %d, want 1", decoded.ReplayData.Entries[0].HostcallID)
	}
}

func TestReplayData_JSONRoundTrip_NilOmitted(t *testing.T) {
	// Package without replay data — should omit the field in JSON
	original := protomsg.AgentPackage{
		AgentID:    "test-agent",
		WASMBinary: []byte{0x00},
		Checkpoint: []byte{0x01},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded protomsg.AgentPackage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.ReplayData != nil {
		t.Error("expected nil ReplayData when not present in JSON")
	}
}

func TestExtractAgentState(t *testing.T) {
	state := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	checkpoint := make([]byte, 17+len(state))
	checkpoint[0] = 0x01
	binary.LittleEndian.PutUint64(checkpoint[1:9], 500000)
	binary.LittleEndian.PutUint64(checkpoint[9:17], 1000)
	copy(checkpoint[17:], state)

	extracted, err := agent.ExtractAgentState(checkpoint)
	if err != nil {
		t.Fatalf("ExtractAgentState: %v", err)
	}
	if len(extracted) != 4 {
		t.Errorf("extracted length: got %d, want 4", len(extracted))
	}
	if extracted[0] != 0xDE || extracted[3] != 0xEF {
		t.Errorf("extracted data mismatch")
	}
}

func TestExtractAgentState_TooShort(t *testing.T) {
	_, err := agent.ExtractAgentState([]byte{0x01, 0x02})
	if err == nil {
		t.Error("expected error for short checkpoint")
	}
}

func TestExtractAgentState_WrongVersion(t *testing.T) {
	checkpoint := make([]byte, 17)
	checkpoint[0] = 0xFF
	_, err := agent.ExtractAgentState(checkpoint)
	if err == nil {
		t.Error("expected error for wrong version")
	}
}

func TestExtractAgentState_V2(t *testing.T) {
	state := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	// v2: [0x02][budget:8][price:8][tickNumber:8][state:4]
	checkpoint := make([]byte, 25+len(state))
	checkpoint[0] = 0x02
	binary.LittleEndian.PutUint64(checkpoint[1:9], 500000)
	binary.LittleEndian.PutUint64(checkpoint[9:17], 1000)
	binary.LittleEndian.PutUint64(checkpoint[17:25], 42) // tickNumber=42
	copy(checkpoint[25:], state)

	extracted, err := agent.ExtractAgentState(checkpoint)
	if err != nil {
		t.Fatalf("ExtractAgentState v2: %v", err)
	}
	if len(extracted) != 4 {
		t.Errorf("extracted length: got %d, want 4", len(extracted))
	}
	if extracted[0] != 0xDE || extracted[3] != 0xEF {
		t.Errorf("extracted data mismatch")
	}
}

func TestParseCheckpointHeader_V1(t *testing.T) {
	state := []byte{0x01, 0x02}
	checkpoint := make([]byte, 17+len(state))
	checkpoint[0] = 0x01
	binary.LittleEndian.PutUint64(checkpoint[1:9], 5000000)
	binary.LittleEndian.PutUint64(checkpoint[9:17], 10000)
	copy(checkpoint[17:], state)

	budgetVal, price, tick, s, err := agent.ParseCheckpointHeader(checkpoint)
	if err != nil {
		t.Fatalf("ParseCheckpointHeader v1: %v", err)
	}
	if budgetVal != 5000000 {
		t.Errorf("budget: got %d, want 5000000", budgetVal)
	}
	if price != 10000 {
		t.Errorf("price: got %d, want 10000", price)
	}
	if tick != 0 {
		t.Errorf("tick: got %d, want 0 (v1 default)", tick)
	}
	if len(s) != 2 || s[0] != 0x01 || s[1] != 0x02 {
		t.Errorf("state mismatch: got %v", s)
	}
}

func TestParseCheckpointHeader_V2(t *testing.T) {
	state := []byte{0xAA, 0xBB}
	checkpoint := make([]byte, 25+len(state))
	checkpoint[0] = 0x02
	binary.LittleEndian.PutUint64(checkpoint[1:9], 7000000)
	binary.LittleEndian.PutUint64(checkpoint[9:17], 20000)
	binary.LittleEndian.PutUint64(checkpoint[17:25], 99)
	copy(checkpoint[25:], state)

	budgetVal, price, tick, s, err := agent.ParseCheckpointHeader(checkpoint)
	if err != nil {
		t.Fatalf("ParseCheckpointHeader v2: %v", err)
	}
	if budgetVal != 7000000 {
		t.Errorf("budget: got %d, want 7000000", budgetVal)
	}
	if price != 20000 {
		t.Errorf("price: got %d, want 20000", price)
	}
	if tick != 99 {
		t.Errorf("tick: got %d, want 99", tick)
	}
	if len(s) != 2 || s[0] != 0xAA || s[1] != 0xBB {
		t.Errorf("state mismatch: got %v", s)
	}
}
