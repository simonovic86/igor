// Package eventlog implements per-tick observation recording for the capability membrane.
// Every observation hostcall (clock, rand, etc.) records its return value in the event log,
// enabling deterministic replay per CM-4 and CE-3.
package eventlog

import "sync"

// HostcallID identifies which hostcall produced an observation entry.
type HostcallID uint16

const (
	ClockNow  HostcallID = 1
	RandBytes HostcallID = 2
	LogEmit   HostcallID = 3
)

// Entry is a single observation recorded during a tick.
type Entry struct {
	HostcallID HostcallID
	Payload    []byte
}

// TickLog holds all observation entries for a single tick execution.
type TickLog struct {
	TickNumber uint64
	Entries    []Entry
}

// DefaultMaxTicks is the maximum number of sealed tick logs retained in history.
// At 1 Hz tick rate this covers ~17 minutes. Use 0 for unbounded (tests only).
const DefaultMaxTicks = 1024

// EventLog records observation hostcall return values per tick.
// It is created once per agent instance and reused across ticks.
type EventLog struct {
	mu       sync.Mutex
	current  *TickLog
	history  []*TickLog
	maxTicks int
}

// NewEventLog creates a new event log. maxTicks bounds the retained history;
// 0 means unbounded (useful for tests).
func NewEventLog(maxTicks int) *EventLog {
	return &EventLog{maxTicks: maxTicks}
}

// BeginTick starts recording for a new tick. Must be called before any
// Record calls for this tick.
func (l *EventLog) BeginTick(tickNumber uint64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.current = &TickLog{
		TickNumber: tickNumber,
		Entries:    nil,
	}
}

// Record appends an observation entry to the current tick log.
// The payload is copied to avoid aliasing with caller buffers.
func (l *EventLog) Record(id HostcallID, payload []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.current == nil {
		return
	}

	p := make([]byte, len(payload))
	copy(p, payload)

	l.current.Entries = append(l.current.Entries, Entry{
		HostcallID: id,
		Payload:    p,
	})
}

// SealTick closes the current tick log and moves it to history.
// Returns the sealed log, or nil if no tick was in progress.
func (l *EventLog) SealTick() *TickLog {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.current == nil {
		return nil
	}

	sealed := l.current
	l.history = append(l.history, sealed)
	l.current = nil

	// Evict oldest ticks when history exceeds maxTicks bound.
	// Copy to a new slice to release references to evicted TickLogs,
	// preventing memory leaks from the retained underlying array.
	if l.maxTicks > 0 && len(l.history) > l.maxTicks {
		kept := l.history[len(l.history)-l.maxTicks:]
		fresh := make([]*TickLog, len(kept))
		copy(fresh, kept)
		l.history = fresh
	}

	return sealed
}

// CurrentEntries returns the entries recorded so far in the current tick.
// Returns nil if no tick is in progress.
func (l *EventLog) CurrentEntries() []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.current == nil {
		return nil
	}
	return l.current.Entries
}

// History returns all sealed tick logs.
func (l *EventLog) History() []*TickLog {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.history
}
