// SPDX-License-Identifier: Apache-2.0

package igor

import (
	"testing"
)

func TestEffectLog_HappyPath(t *testing.T) {
	var e EffectLog

	id := []byte("tx-001")
	data := []byte(`{"amount":100}`)

	if err := e.Record(id, data); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if e.Len() != 1 {
		t.Fatalf("expected 1 intent, got %d", e.Len())
	}

	intent := e.Get(id)
	if intent == nil {
		t.Fatal("Get returned nil")
	}
	if intent.State != Recorded {
		t.Fatalf("expected Recorded, got %d", intent.State)
	}

	if err := e.Begin(id); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if intent.State != InFlight {
		t.Fatalf("expected InFlight, got %d", intent.State)
	}

	if err := e.Confirm(id); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if intent.State != Confirmed {
		t.Fatalf("expected Confirmed, got %d", intent.State)
	}

	// Terminal — should not appear in Pending.
	if len(e.Pending()) != 0 {
		t.Fatalf("expected 0 pending, got %d", len(e.Pending()))
	}
}

func TestEffectLog_DuplicateID(t *testing.T) {
	var e EffectLog
	id := []byte("dup")
	if err := e.Record(id, nil); err != nil {
		t.Fatal(err)
	}
	if err := e.Record(id, nil); err == nil {
		t.Fatal("expected error on duplicate ID")
	}
}

func TestEffectLog_InvalidTransitions(t *testing.T) {
	var e EffectLog
	id := []byte("tx")
	e.Record(id, nil)

	// Can't Confirm from Recorded.
	if err := e.Confirm(id); err == nil {
		t.Fatal("expected error: Confirm from Recorded")
	}
	// Can't Compensate from Recorded.
	if err := e.Compensate(id); err == nil {
		t.Fatal("expected error: Compensate from Recorded")
	}

	e.Begin(id)

	// Can't Begin again.
	if err := e.Begin(id); err == nil {
		t.Fatal("expected error: Begin from InFlight")
	}

	e.Confirm(id)

	// Can't do anything from Confirmed.
	if err := e.Begin(id); err == nil {
		t.Fatal("expected error: Begin from Confirmed")
	}
	if err := e.Confirm(id); err == nil {
		t.Fatal("expected error: Confirm from Confirmed")
	}
}

func TestEffectLog_ResumeRule(t *testing.T) {
	var e EffectLog

	e.Record([]byte("recorded"), nil)
	e.Record([]byte("inflight"), nil)
	e.Begin([]byte("inflight"))
	e.Record([]byte("confirmed"), nil)
	e.Begin([]byte("confirmed"))
	e.Confirm([]byte("confirmed"))

	// Serialize and deserialize (simulates checkpoint → crash → resume).
	data := e.Marshal()

	var resumed EffectLog
	resumed.Unmarshal(data)

	// "recorded" should stay Recorded.
	if intent := resumed.Get([]byte("recorded")); intent.State != Recorded {
		t.Fatalf("expected Recorded, got %d", intent.State)
	}

	// "inflight" should become Unresolved (THE RESUME RULE).
	if intent := resumed.Get([]byte("inflight")); intent.State != Unresolved {
		t.Fatalf("expected Unresolved, got %d", intent.State)
	}

	// "confirmed" should stay Confirmed.
	if intent := resumed.Get([]byte("confirmed")); intent.State != Confirmed {
		t.Fatalf("expected Confirmed, got %d", intent.State)
	}

	// Unresolved list should have exactly one.
	unresolved := resumed.Unresolved()
	if len(unresolved) != 1 {
		t.Fatalf("expected 1 unresolved, got %d", len(unresolved))
	}
	if string(unresolved[0].ID) != "inflight" {
		t.Fatalf("wrong unresolved ID: %s", unresolved[0].ID)
	}
}

func TestEffectLog_Compensate(t *testing.T) {
	var e EffectLog
	id := []byte("tx")
	e.Record(id, []byte("payload"))
	e.Begin(id)

	// Simulate crash: marshal while InFlight, unmarshal.
	data := e.Marshal()
	var resumed EffectLog
	resumed.Unmarshal(data)

	intent := resumed.Get(id)
	if intent.State != Unresolved {
		t.Fatalf("expected Unresolved, got %d", intent.State)
	}

	if err := resumed.Compensate(id); err != nil {
		t.Fatalf("Compensate: %v", err)
	}
	if intent.State != Compensated {
		t.Fatalf("expected Compensated, got %d", intent.State)
	}

	// Compensated is terminal.
	if err := resumed.Confirm(id); err == nil {
		t.Fatal("expected error: Confirm from Compensated")
	}
}

func TestEffectLog_CompensateFromInFlight(t *testing.T) {
	var e EffectLog
	id := []byte("tx")
	e.Record(id, nil)
	e.Begin(id)

	// Agent observes the action fail (e.g., HTTP error, payment rejected).
	// It knows the action didn't complete, so it compensates directly.
	if err := e.Compensate(id); err != nil {
		t.Fatalf("Compensate from InFlight: %v", err)
	}
	if e.Get(id).State != Compensated {
		t.Fatal("expected Compensated")
	}

	// Compensated is terminal — prune should remove it.
	removed := e.Prune()
	if removed != 1 {
		t.Fatalf("expected 1 pruned, got %d", removed)
	}
}

func TestEffectLog_ConfirmFromUnresolved(t *testing.T) {
	var e EffectLog
	id := []byte("tx")
	e.Record(id, nil)
	e.Begin(id)

	// Simulate crash.
	data := e.Marshal()
	var resumed EffectLog
	resumed.Unmarshal(data)

	// Agent checks external state, finds the action completed.
	if err := resumed.Confirm(id); err != nil {
		t.Fatalf("Confirm from Unresolved: %v", err)
	}
	if resumed.Get(id).State != Confirmed {
		t.Fatal("expected Confirmed")
	}
}

func TestEffectLog_Prune(t *testing.T) {
	var e EffectLog
	e.Record([]byte("a"), nil)
	e.Begin([]byte("a"))
	e.Confirm([]byte("a"))

	e.Record([]byte("b"), nil)

	e.Record([]byte("c"), nil)
	e.Begin([]byte("c"))

	// Simulate crash for c.
	data := e.Marshal()
	var resumed EffectLog
	resumed.Unmarshal(data)
	resumed.Compensate([]byte("c"))

	// a=Confirmed, b=Recorded, c=Compensated
	removed := resumed.Prune()
	if removed != 2 {
		t.Fatalf("expected 2 pruned, got %d", removed)
	}
	if resumed.Len() != 1 {
		t.Fatalf("expected 1 remaining, got %d", resumed.Len())
	}
	if resumed.Get([]byte("b")) == nil {
		t.Fatal("expected b to survive prune")
	}
}

func TestEffectLog_Pending(t *testing.T) {
	var e EffectLog
	e.Record([]byte("r"), nil)
	e.Record([]byte("i"), nil)
	e.Begin([]byte("i"))
	e.Record([]byte("c"), nil)
	e.Begin([]byte("c"))
	e.Confirm([]byte("c"))

	pending := e.Pending()
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(pending))
	}
}

func TestEffectLog_MultipleIntents(t *testing.T) {
	var e EffectLog

	// Agent monitoring multiple positions.
	for i := 0; i < 5; i++ {
		id := []byte{byte(i)}
		if err := e.Record(id, nil); err != nil {
			t.Fatalf("Record %d: %v", i, err)
		}
	}
	if e.Len() != 5 {
		t.Fatalf("expected 5, got %d", e.Len())
	}

	// Begin all.
	for i := 0; i < 5; i++ {
		e.Begin([]byte{byte(i)})
	}

	// Crash and resume.
	data := e.Marshal()
	var resumed EffectLog
	resumed.Unmarshal(data)

	// All should be Unresolved.
	if len(resumed.Unresolved()) != 5 {
		t.Fatalf("expected 5 unresolved, got %d", len(resumed.Unresolved()))
	}
}

func TestEffectLog_EmptyMarshal(t *testing.T) {
	var e EffectLog
	data := e.Marshal()
	var resumed EffectLog
	resumed.Unmarshal(data)
	if resumed.Len() != 0 {
		t.Fatalf("expected 0, got %d", resumed.Len())
	}
}

func TestEffectLog_DataPreserved(t *testing.T) {
	var e EffectLog
	id := []byte("tx")
	payload := []byte(`{"chain":"ethereum","amount":"1.5","to":"0xabc"}`)
	e.Record(id, payload)
	e.Begin(id)

	data := e.Marshal()
	var resumed EffectLog
	resumed.Unmarshal(data)

	intent := resumed.Get(id)
	if string(intent.Data) != string(payload) {
		t.Fatalf("data not preserved: got %q", intent.Data)
	}
}

func TestEffectLog_NotFoundErrors(t *testing.T) {
	var e EffectLog
	missing := []byte("nope")

	if err := e.Begin(missing); err == nil {
		t.Fatal("expected error for Begin on missing intent")
	}
	if err := e.Confirm(missing); err == nil {
		t.Fatal("expected error for Confirm on missing intent")
	}
	if err := e.Compensate(missing); err == nil {
		t.Fatal("expected error for Compensate on missing intent")
	}
	if e.Get(missing) != nil {
		t.Fatal("expected nil for Get on missing intent")
	}
}
