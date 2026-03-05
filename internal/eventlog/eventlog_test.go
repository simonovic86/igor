// SPDX-License-Identifier: Apache-2.0

package eventlog

import (
	"bytes"
	"testing"
)

func TestEventLog_BeginRecordSeal(t *testing.T) {
	el := NewEventLog(0)

	el.BeginTick(1)
	el.Record(ClockNow, []byte{0x01, 0x02})
	el.Record(RandBytes, []byte{0xAA, 0xBB, 0xCC})

	entries := el.CurrentEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].HostcallID != ClockNow {
		t.Errorf("expected ClockNow, got %d", entries[0].HostcallID)
	}
	if entries[1].HostcallID != RandBytes {
		t.Errorf("expected RandBytes, got %d", entries[1].HostcallID)
	}

	sealed := el.SealTick()
	if sealed == nil {
		t.Fatal("expected sealed tick log")
	}
	if sealed.TickNumber != 1 {
		t.Errorf("expected tick 1, got %d", sealed.TickNumber)
	}
	if len(sealed.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(sealed.Entries))
	}
}

func TestEventLog_NoTickInProgress(t *testing.T) {
	el := NewEventLog(0)

	// Record without BeginTick should be a no-op
	el.Record(ClockNow, []byte{0x01})

	entries := el.CurrentEntries()
	if entries != nil {
		t.Errorf("expected nil entries, got %v", entries)
	}

	sealed := el.SealTick()
	if sealed != nil {
		t.Errorf("expected nil sealed, got %v", sealed)
	}
}

func TestEventLog_MultipleTicks(t *testing.T) {
	el := NewEventLog(0)

	el.BeginTick(1)
	el.Record(ClockNow, []byte{0x01})
	el.SealTick()

	el.BeginTick(2)
	el.Record(RandBytes, []byte{0x02})
	el.Record(LogEmit, []byte{0x03})
	el.SealTick()

	history := el.History()
	if len(history) != 2 {
		t.Fatalf("expected 2 ticks in history, got %d", len(history))
	}
	if history[0].TickNumber != 1 {
		t.Errorf("expected tick 1, got %d", history[0].TickNumber)
	}
	if len(history[0].Entries) != 1 {
		t.Errorf("expected 1 entry in tick 1, got %d", len(history[0].Entries))
	}
	if history[1].TickNumber != 2 {
		t.Errorf("expected tick 2, got %d", history[1].TickNumber)
	}
	if len(history[1].Entries) != 2 {
		t.Errorf("expected 2 entries in tick 2, got %d", len(history[1].Entries))
	}
}

func TestEventLog_PayloadCopied(t *testing.T) {
	el := NewEventLog(0)
	el.BeginTick(1)

	buf := []byte{0x01, 0x02, 0x03}
	el.Record(ClockNow, buf)

	// Mutate the original buffer
	buf[0] = 0xFF

	entries := el.CurrentEntries()
	if bytes.Equal(entries[0].Payload, buf) {
		t.Error("payload should be a copy, not alias the original buffer")
	}
	if entries[0].Payload[0] != 0x01 {
		t.Errorf("expected 0x01, got 0x%02X", entries[0].Payload[0])
	}
}

func TestEventLog_EmptyTick(t *testing.T) {
	el := NewEventLog(0)
	el.BeginTick(1)
	sealed := el.SealTick()

	if sealed == nil {
		t.Fatal("expected sealed tick log even with no entries")
	}
	if len(sealed.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(sealed.Entries))
	}
}

func TestEventLog_Eviction(t *testing.T) {
	el := NewEventLog(3)

	// Seal 5 ticks; only the last 3 should be retained
	for i := uint64(1); i <= 5; i++ {
		el.BeginTick(i)
		el.Record(ClockNow, []byte{byte(i)})
		el.SealTick()
	}

	history := el.History()
	if len(history) != 3 {
		t.Fatalf("expected 3 ticks in history, got %d", len(history))
	}
	if history[0].TickNumber != 3 {
		t.Errorf("oldest tick: got %d, want 3", history[0].TickNumber)
	}
	if history[2].TickNumber != 5 {
		t.Errorf("newest tick: got %d, want 5", history[2].TickNumber)
	}
}

func TestEventLog_EvictionReleasesMemory(t *testing.T) {
	el := NewEventLog(3)
	for i := uint64(1); i <= 10; i++ {
		el.BeginTick(i)
		el.Record(ClockNow, make([]byte, 1024))
		el.SealTick()
	}
	history := el.History()
	if len(history) != 3 {
		t.Fatalf("expected 3, got %d", len(history))
	}
	// Verify capacity is not leaking (should be exactly 3, not 10)
	if cap(history) > 3 {
		t.Errorf("history capacity should be 3, got %d (memory leak)", cap(history))
	}
}

func TestEventLog_ArenaFallback(t *testing.T) {
	el := NewEventLog(0)
	el.BeginTick(1)

	// Record entries that exceed the default arena size (4096 bytes).
	// Each entry is 1024 bytes, so after 4 entries the arena is full and
	// subsequent entries fall back to heap allocation.
	for i := 0; i < 6; i++ {
		payload := make([]byte, 1024)
		payload[0] = byte(i)
		el.Record(ClockNow, payload)
	}

	entries := el.CurrentEntries()
	if len(entries) != 6 {
		t.Fatalf("expected 6 entries, got %d", len(entries))
	}

	// Verify all payloads are correct regardless of arena vs heap backing.
	for i, e := range entries {
		if e.Payload[0] != byte(i) {
			t.Errorf("entry %d: expected first byte %d, got %d", i, i, e.Payload[0])
		}
		if len(e.Payload) != 1024 {
			t.Errorf("entry %d: expected 1024 bytes, got %d", i, len(e.Payload))
		}
	}
}

func TestEventLog_UnboundedWhenZero(t *testing.T) {
	el := NewEventLog(0)

	for i := uint64(1); i <= 100; i++ {
		el.BeginTick(i)
		el.SealTick()
	}

	history := el.History()
	if len(history) != 100 {
		t.Errorf("expected 100 ticks in history, got %d", len(history))
	}
}
