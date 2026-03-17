// SPDX-License-Identifier: Apache-2.0

package igor

import (
	"bytes"
	"errors"
)

// IntentState represents the lifecycle state of an effect intent.
// See docs/runtime/EFFECT_LIFECYCLE.md for the full state machine.
type IntentState uint8

const (
	// Recorded means the agent has declared intent to perform an external action.
	// The intent exists in state and will survive checkpoint, but no action has been attempted.
	Recorded IntentState = 1

	// InFlight means the agent has begun the external action.
	// If a crash occurs, this becomes Unresolved on resume.
	InFlight IntentState = 2

	// Confirmed means the external action was verified complete. Terminal.
	Confirmed IntentState = 3

	// Unresolved means a crash occurred while the intent was InFlight.
	// The agent does not know whether the action completed.
	// Must be resolved via Confirm (it happened) or Compensate (it didn't).
	Unresolved IntentState = 4

	// Compensated means the agent determined an Unresolved intent either
	// did not execute or has been rolled back. Terminal.
	Compensated IntentState = 5
)

// Intent is an individual effect intent tracked by the EffectLog.
type Intent struct {
	ID    []byte
	State IntentState
	Data  []byte // Agent-defined payload (tx params, API call details, etc.)
}

var (
	errIntentExists      = errors.New("igor: intent ID already exists")
	errIntentNotFound    = errors.New("igor: intent not found")
	errInvalidTransition = errors.New("igor: invalid intent state transition")
)

// EffectLog tracks intents across checkpoint/resume cycles.
// Embed this in your agent struct and include it in Marshal/Unmarshal.
type EffectLog struct {
	intents []Intent
}

// Record declares a new intent. The intent starts in Recorded state.
// Returns an error if an intent with the same ID already exists.
func (e *EffectLog) Record(id, data []byte) error {
	if e.find(id) != nil {
		return errIntentExists
	}
	e.intents = append(e.intents, Intent{
		ID:    copyBytes(id),
		State: Recorded,
		Data:  copyBytes(data),
	})
	return nil
}

// Begin transitions an intent from Recorded to InFlight.
// Call this immediately before performing the external action.
func (e *EffectLog) Begin(id []byte) error {
	intent := e.find(id)
	if intent == nil {
		return errIntentNotFound
	}
	if intent.State != Recorded {
		return errInvalidTransition
	}
	intent.State = InFlight
	return nil
}

// Confirm transitions an intent from InFlight or Unresolved to Confirmed.
// Call this after verifying the external action succeeded.
func (e *EffectLog) Confirm(id []byte) error {
	intent := e.find(id)
	if intent == nil {
		return errIntentNotFound
	}
	if intent.State != InFlight && intent.State != Unresolved {
		return errInvalidTransition
	}
	intent.State = Confirmed
	return nil
}

// Compensate transitions an intent from InFlight or Unresolved to Compensated.
// Call this after determining the action did not happen or was rolled back.
// From InFlight: the agent observed the action fail (e.g., HTTP error, payment rejected).
// From Unresolved: the agent investigated after a crash and determined the action didn't complete.
func (e *EffectLog) Compensate(id []byte) error {
	intent := e.find(id)
	if intent == nil {
		return errIntentNotFound
	}
	if intent.State != InFlight && intent.State != Unresolved {
		return errInvalidTransition
	}
	intent.State = Compensated
	return nil
}

// Pending returns all intents not in a terminal state (Recorded, InFlight, Unresolved).
func (e *EffectLog) Pending() []Intent {
	var out []Intent
	for _, intent := range e.intents {
		if intent.State != Confirmed && intent.State != Compensated {
			out = append(out, intent)
		}
	}
	return out
}

// Unresolved returns only intents in Unresolved state.
// Check this on resume to handle the crash recovery swamp.
func (e *EffectLog) Unresolved() []Intent {
	var out []Intent
	for _, intent := range e.intents {
		if intent.State == Unresolved {
			out = append(out, intent)
		}
	}
	return out
}

// Get returns the intent with the given ID, or nil if not found.
func (e *EffectLog) Get(id []byte) *Intent {
	return e.find(id)
}

// Len returns the total number of intents (all states).
func (e *EffectLog) Len() int {
	return len(e.intents)
}

// Prune removes all terminal intents (Confirmed, Compensated).
// Call periodically to prevent unbounded growth.
// Returns the number of intents removed.
func (e *EffectLog) Prune() int {
	n := 0
	kept := e.intents[:0]
	for _, intent := range e.intents {
		if intent.State == Confirmed || intent.State == Compensated {
			n++
		} else {
			kept = append(kept, intent)
		}
	}
	e.intents = kept
	return n
}

// Marshal serializes the effect log for checkpointing.
// Wire format (little-endian):
//
//	[count: uint32]
//	  for each intent:
//	    [state: uint8]
//	    [id_len: uint32][id: N bytes]
//	    [data_len: uint32][data: N bytes]
func (e *EffectLog) Marshal() []byte {
	// Estimate size: 4 + N * (1 + 4 + avg_id + 4 + avg_data)
	enc := NewEncoder(4 + len(e.intents)*32)
	enc.Uint32(uint32(len(e.intents)))
	for _, intent := range e.intents {
		enc.buf = append(enc.buf, byte(intent.State))
		enc.Bytes(intent.ID)
		enc.Bytes(intent.Data)
	}
	return enc.Finish()
}

// Unmarshal restores the effect log from serialized bytes.
// CRITICAL: All InFlight intents become Unresolved. This is the resume rule.
func (e *EffectLog) Unmarshal(data []byte) {
	if len(data) == 0 {
		e.intents = nil
		return
	}
	d := NewDecoder(data)
	count := d.Uint32()
	if d.Err() != nil {
		e.intents = nil
		return
	}
	e.intents = make([]Intent, 0, count)
	for i := uint32(0); i < count; i++ {
		if d.Err() != nil {
			break
		}
		if d.pos >= len(d.data) {
			break
		}
		state := IntentState(d.data[d.pos])
		d.pos++

		id := d.Bytes()
		payload := d.Bytes()

		// THE RESUME RULE: InFlight → Unresolved
		if state == InFlight {
			state = Unresolved
		}

		e.intents = append(e.intents, Intent{
			ID:    id,
			State: state,
			Data:  payload,
		})
	}
}

// find returns a pointer to the intent with the given ID, or nil.
func (e *EffectLog) find(id []byte) *Intent {
	for i := range e.intents {
		if bytes.Equal(e.intents[i].ID, id) {
			return &e.intents[i]
		}
	}
	return nil
}

// copyBytes returns a copy of b, or nil if b is nil.
func copyBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
