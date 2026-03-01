package migration

import (
	"bytes"

	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/internal/eventlog"
	protomsg "github.com/simonovic86/igor/pkg/protocol"
)

// replayDataFromInstance extracts replay data from a live agent instance.
// Returns nil if no tick has been executed yet.
// The checkpoint parameter is the stored checkpoint; replay data is only
// included when the latest snapshot's PostState matches the checkpoint's
// state portion (staleness guard — ensures replay data corresponds to the
// stored checkpoint).
func replayDataFromInstance(inst *agent.Instance, checkpoint []byte) *protomsg.ReplayData {
	snap := inst.LatestSnapshot()
	if snap == nil || snap.TickLog == nil {
		return nil
	}

	// Staleness guard: only include replay data when PostState matches
	// the checkpoint we're sending. If more ticks occurred after the last
	// checkpoint save, the replay data would not correspond to the checkpoint.
	storedState, err := agent.ExtractAgentState(checkpoint)
	if err != nil {
		return nil
	}
	if !bytes.Equal(snap.PostState, storedState) {
		return nil
	}

	entries := make([]protomsg.ReplayEntry, len(snap.TickLog.Entries))
	for i, e := range snap.TickLog.Entries {
		entries[i] = protomsg.ReplayEntry{
			HostcallID: uint16(e.HostcallID),
			Payload:    e.Payload,
		}
	}
	return &protomsg.ReplayData{
		PreTickState: snap.PreState,
		TickNumber:   snap.TickLog.TickNumber,
		Entries:      entries,
	}
}

// toTickLog converts protocol ReplayData to an eventlog.TickLog
// for use with the replay engine.
func toTickLog(rd *protomsg.ReplayData) *eventlog.TickLog {
	entries := make([]eventlog.Entry, len(rd.Entries))
	for i, e := range rd.Entries {
		entries[i] = eventlog.Entry{
			HostcallID: eventlog.HostcallID(e.HostcallID),
			Payload:    e.Payload,
		}
	}
	return &eventlog.TickLog{
		TickNumber: rd.TickNumber,
		Entries:    entries,
	}
}
