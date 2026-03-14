// SPDX-License-Identifier: Apache-2.0

//go:build tinygo || wasip1

package main

import (
	"encoding/hex"

	"github.com/simonovic86/igor/sdk/igor"
)

// State machine states for bridge reconciliation.
const (
	StateDetectedPendingTransfer uint32 = 0
	StateWaitingConfirmations    uint32 = 1
	StateReadyToFinalize         uint32 = 2
	StateFinalized               uint32 = 3
	StateCompleted               uint32 = 4
)

// requiredConfirmations is the simulated block confirmation threshold.
const requiredConfirmations uint32 = 6

// Reconciler implements a bridge reconciliation agent that detects pending
// cross-chain transfers, waits for confirmations, executes a finalize side
// effect with idempotency tracking, and completes the case.
type Reconciler struct {
	State            uint32
	TickCount        uint64
	CaseID           [16]byte // Random case identifier
	Confirmations    uint32   // Current confirmation count
	RequiredConfs    uint32   // Confirmation threshold
	IdempotencyKey   [16]byte // Finalize idempotency key
	IntentRecorded   bool     // Finalize intent captured in checkpoint
	FinalizeExecuted bool     // Finalize side effect performed
	AckReceived      bool     // External acknowledgment received
	DupChecked       bool     // Post-resume duplicate check done
	DetectNano       int64    // Timestamp of case detection
	FinalizeNano     int64    // Timestamp of finalize execution
	CompleteNano     int64    // Timestamp of completion
}

func (r *Reconciler) Init() {}

func (r *Reconciler) Tick() bool {
	r.TickCount++
	now := igor.ClockNow()

	switch r.State {
	case StateDetectedPendingTransfer:
		return r.tickDetected(now)
	case StateWaitingConfirmations:
		return r.tickWaitingConfirmations(now)
	case StateReadyToFinalize:
		return r.tickReadyToFinalize(now)
	case StateFinalized:
		return r.tickFinalized(now)
	case StateCompleted:
		return r.tickCompleted(now)
	}
	return false
}

func (r *Reconciler) tickDetected(now int64) bool {
	// First tick: generate case ID and detect the pending transfer.
	r.DetectNano = now
	_ = igor.RandBytes(r.CaseID[:])
	r.RequiredConfs = requiredConfirmations

	caseHex := shortHex(r.CaseID[:])
	igor.Logf("[reconciler] Case %s detected (source: chain-A, dest: chain-B)", caseHex)
	igor.Logf("[reconciler] State: DetectedPendingTransfer")

	r.State = StateWaitingConfirmations
	return true
}

func (r *Reconciler) tickWaitingConfirmations(now int64) bool {
	r.Confirmations++
	caseHex := shortHex(r.CaseID[:])

	igor.Logf("[reconciler] Confirmation %d/%d for case %s",
		r.Confirmations, r.RequiredConfs, caseHex)

	if r.Confirmations >= r.RequiredConfs {
		// Threshold met: generate idempotency key and transition.
		_ = igor.RandBytes(r.IdempotencyKey[:])
		keyHex := shortHex(r.IdempotencyKey[:])

		igor.Logf("[reconciler] Confirmation threshold met (%d/%d blocks)",
			r.RequiredConfs, r.RequiredConfs)
		igor.Logf("[reconciler] State: WaitingConfirmations -> ReadyToFinalize")
		igor.Logf("[reconciler] Idempotency key generated: %s", keyHex)

		r.State = StateReadyToFinalize
	}
	return true
}

func (r *Reconciler) tickReadyToFinalize(now int64) bool {
	caseHex := shortHex(r.CaseID[:])
	keyHex := shortHex(r.IdempotencyKey[:])

	if !r.IntentRecorded {
		// Record finalize intent. This will be captured in the next checkpoint.
		r.IntentRecorded = true
		igor.Logf("[reconciler] Finalize intent recorded")
		igor.Logf("[reconciler] Idempotency key: %s", keyHex)
		return true
	}

	if !r.FinalizeExecuted {
		// Execute the finalize side effect.
		r.FinalizeExecuted = true
		r.FinalizeNano = now
		igor.Logf("[reconciler] Finalize EXECUTED for case %s (key: %s)", caseHex, keyHex)
		igor.Logf("[reconciler] State: ReadyToFinalize -> Finalized")
		r.State = StateFinalized
		return true
	}

	return true
}

func (r *Reconciler) tickFinalized(now int64) bool {
	keyHex := shortHex(r.IdempotencyKey[:])

	if !r.DupChecked {
		// First tick in Finalized state after resume: check for duplicate.
		r.DupChecked = true

		if r.FinalizeExecuted {
			igor.Logf("[reconciler] Finalize action SKIPPED")
			igor.Logf("[reconciler] Reason: intent %s already committed", keyHex)
			igor.Logf("[reconciler] No duplicate execution")
		}
		return true
	}

	if !r.AckReceived {
		// Simulate receiving acknowledgment from destination chain.
		r.AckReceived = true
		igor.Logf("[reconciler] Acknowledgment received from chain-B")
		igor.Logf("[reconciler] State: Finalized -> Completed")
		r.State = StateCompleted
		return true
	}

	return true
}

func (r *Reconciler) tickCompleted(now int64) bool {
	r.CompleteNano = now
	caseHex := shortHex(r.CaseID[:])

	totalDowntime := int64(0)
	if r.FinalizeNano > 0 && r.CompleteNano > r.FinalizeNano {
		totalDowntime = (r.CompleteNano - r.FinalizeNano) / 1_000_000_000
	}

	igor.Logf("[reconciler] Case %s completed", caseHex)
	igor.Logf("[reconciler] Side effects executed: 1 (exactly once)")
	igor.Logf("[reconciler] Total processing time: %ds", totalDowntime)
	igor.Logf("[reconciler] State: Completed")
	return false
}

// Marshal serializes the agent state for checkpointing.
// Pure function — no hostcalls (CM-4 compliance).
func (r *Reconciler) Marshal() []byte {
	return igor.NewEncoder(128).
		Uint32(r.State).
		Uint64(r.TickCount).
		Bytes(r.CaseID[:]).
		Uint32(r.Confirmations).
		Uint32(r.RequiredConfs).
		Bytes(r.IdempotencyKey[:]).
		Bool(r.IntentRecorded).
		Bool(r.FinalizeExecuted).
		Bool(r.AckReceived).
		Bool(r.DupChecked).
		Int64(r.DetectNano).
		Int64(r.FinalizeNano).
		Int64(r.CompleteNano).
		Finish()
}

// Unmarshal restores the agent state from a checkpoint.
// Pure function — no hostcalls (CM-4 compliance).
func (r *Reconciler) Unmarshal(data []byte) {
	d := igor.NewDecoder(data)
	r.State = d.Uint32()
	r.TickCount = d.Uint64()
	copy(r.CaseID[:], d.Bytes())
	r.Confirmations = d.Uint32()
	r.RequiredConfs = d.Uint32()
	copy(r.IdempotencyKey[:], d.Bytes())
	r.IntentRecorded = d.Bool()
	r.FinalizeExecuted = d.Bool()
	r.AckReceived = d.Bool()
	r.DupChecked = d.Bool()
	r.DetectNano = d.Int64()
	r.FinalizeNano = d.Int64()
	r.CompleteNano = d.Int64()
	if err := d.Err(); err != nil {
		panic("unmarshal checkpoint: " + err.Error())
	}
}

// shortHex returns the first 8 hex chars of a byte slice (4 bytes).
func shortHex(b []byte) string {
	if len(b) < 4 {
		return hex.EncodeToString(b)
	}
	return hex.EncodeToString(b[:4])
}

func init() { igor.Run(&Reconciler{}) }
func main() {}
