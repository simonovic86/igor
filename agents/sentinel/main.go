// SPDX-License-Identifier: Apache-2.0

//go:build tinygo || wasip1

// Treasury Sentinel: a DeFi operator that monitors a treasury balance,
// triggers refills when it drops below threshold, and handles crash
// recovery using Igor's effect model.
//
// The world is simulated internally — no real chain dependency. The point
// is to demonstrate the effect lifecycle under partial failure:
//   - Intent is recorded before any external action
//   - Checkpoint captures the intent
//   - If crashed mid-flight, resume discovers Unresolved intents
//   - Agent reconciles by checking simulated bridge status
//   - No blind retry, no duplicate transfer
package main

import (
	"encoding/binary"

	"github.com/simonovic86/igor/sdk/igor"
)

// Treasury simulation parameters.
const (
	initialBalance  uint64 = 10_000_00 // $10,000.00 in cents
	refillThreshold uint64 = 3_000_00  // refill when below $3,000
	refillAmount    uint64 = 5_000_00  // refill $5,000 each time
	spendMin        uint64 = 50_00     // min spend per tick: $50
	spendRange      uint64 = 200_00    // spend variation: up to $200 extra
)

// Sentinel phases.
const (
	phaseMonitoring  uint8 = 0 // watching balance
	phaseReconciling uint8 = 1 // handling unresolved intents on resume
)

// Sentinel implements a treasury monitoring agent with effect-safe refills.
type Sentinel struct {
	TickCount uint64
	BirthNano int64
	LastNano  int64

	// Treasury state (simulated).
	Balance     uint64 // current treasury balance in cents
	RefillCount uint32 // successful refills completed
	SpendTotal  uint64 // total spent across lifetime

	// Effect tracking.
	Effects igor.EffectLog
	Phase   uint8

	// RNG state for deterministic simulation (seeded from igor.RandBytes).
	rngState uint64
}

func (s *Sentinel) Init() {}

func (s *Sentinel) Tick() bool {
	s.TickCount++
	now := igor.ClockNow()
	if s.BirthNano == 0 {
		s.BirthNano = now
		s.Balance = initialBalance
		// Seed RNG.
		var seed [8]byte
		_ = igor.RandBytes(seed[:])
		s.rngState = binary.LittleEndian.Uint64(seed[:])
		igor.Logf("[sentinel] Treasury initialized: $%d.%02d", s.Balance/100, s.Balance%100)
	}
	s.LastNano = now
	ageSec := (s.LastNano - s.BirthNano) / 1_000_000_000

	// On resume, handle unresolved intents first.
	unresolved := s.Effects.Unresolved()
	if len(unresolved) > 0 {
		s.Phase = phaseReconciling
		for _, intent := range unresolved {
			s.reconcile(intent, ageSec)
		}
		s.Effects.Prune()
		s.Phase = phaseMonitoring
		return true
	}

	// Check for pending intents that need execution.
	pending := s.Effects.Pending()
	for _, intent := range pending {
		if intent.State == igor.Recorded {
			return s.executeRefill(intent, ageSec)
		}
	}

	// Normal operation: simulate spending.
	spend := spendMin + (s.rng() % spendRange)
	if spend > s.Balance {
		spend = s.Balance
	}
	s.Balance -= spend
	s.SpendTotal += spend

	// Log every 5 ticks.
	if s.TickCount%5 == 0 {
		igor.Logf("[sentinel] tick=%d age=%ds balance=$%d.%02d spent=$%d.%02d refills=%d",
			s.TickCount, ageSec,
			s.Balance/100, s.Balance%100,
			s.SpendTotal/100, s.SpendTotal%100,
			s.RefillCount)
	}

	// Check if refill needed.
	if s.Balance < refillThreshold {
		igor.Logf("[sentinel] ALERT: balance $%d.%02d below threshold $%d.%02d — initiating refill",
			s.Balance/100, s.Balance%100,
			refillThreshold/100, refillThreshold%100)
		return s.recordRefillIntent(ageSec)
	}

	return false
}

// recordRefillIntent creates a new refill intent. The intent is captured
// in the next checkpoint. The actual transfer happens in a subsequent tick.
func (s *Sentinel) recordRefillIntent(ageSec int64) bool {
	// Generate idempotency key.
	var key [16]byte
	_ = igor.RandBytes(key[:])

	// Encode transfer parameters.
	data := igor.NewEncoder(32).
		Uint64(refillAmount).
		Uint64(s.Balance).
		Int64(ageSec).
		Finish()

	if err := s.Effects.Record(key[:], data); err != nil {
		igor.Logf("[sentinel] ERROR: failed to record intent: %s", err.Error())
		return false
	}

	igor.Logf("[sentinel] Intent RECORDED: refill $%d.%02d (key=%x...)",
		refillAmount/100, refillAmount%100, key[:4])
	igor.Logf("[sentinel] Waiting for checkpoint before execution...")
	return true // request fast tick so we proceed quickly
}

// executeRefill transitions a Recorded intent to InFlight and performs
// the simulated bridge transfer.
func (s *Sentinel) executeRefill(intent igor.Intent, ageSec int64) bool {
	if err := s.Effects.Begin(intent.ID); err != nil {
		igor.Logf("[sentinel] ERROR: failed to begin intent: %s", err.Error())
		return false
	}

	igor.Logf("[sentinel] Refill IN-FLIGHT: executing bridge transfer (key=%x...)", intent.ID[:4])
	igor.Logf("[sentinel] Bridge: source=chain-A dest=chain-B amount=$%d.%02d",
		refillAmount/100, refillAmount%100)

	// === THIS IS THE DANGER ZONE ===
	// If the process crashes between Begin and Confirm, the intent becomes
	// Unresolved on resume. The agent must then reconcile.

	// Simulate the bridge transfer succeeding.
	s.Balance += refillAmount
	s.RefillCount++

	if err := s.Effects.Confirm(intent.ID); err != nil {
		igor.Logf("[sentinel] ERROR: failed to confirm intent: %s", err.Error())
		return false
	}

	igor.Logf("[sentinel] Refill CONFIRMED: balance now $%d.%02d (key=%x...)",
		s.Balance/100, s.Balance%100, intent.ID[:4])

	s.Effects.Prune()
	return true
}

// reconcile handles an unresolved intent after crash recovery.
// In a real system, this would query the bridge API to check if the
// transfer completed. Here we simulate: if the idempotency key's first
// byte is even, the bridge completed the transfer before the crash.
func (s *Sentinel) reconcile(intent igor.Intent, ageSec int64) {
	igor.Logf("[sentinel] RECONCILING: found unresolved intent (key=%x...)", intent.ID[:4])
	igor.Logf("[sentinel] Checking bridge status for in-flight transfer...")

	// Simulate bridge status check.
	// In production: HTTP call to bridge API with idempotency key.
	bridgeCompleted := (intent.ID[0] % 2) == 0

	if bridgeCompleted {
		// The transfer happened before the crash. Credit the balance,
		// but do NOT re-execute the transfer.
		s.Balance += refillAmount
		s.RefillCount++

		if err := s.Effects.Confirm(intent.ID); err != nil {
			igor.Logf("[sentinel] ERROR: reconcile confirm failed: %s", err.Error())
			return
		}

		igor.Logf("[sentinel] Reconciled: transfer COMPLETED before crash")
		igor.Logf("[sentinel] Balance credited: $%d.%02d (no duplicate execution)",
			s.Balance/100, s.Balance%100)
	} else {
		// The transfer did not complete. Safe to compensate and retry.
		if err := s.Effects.Compensate(intent.ID); err != nil {
			igor.Logf("[sentinel] ERROR: reconcile compensate failed: %s", err.Error())
			return
		}

		igor.Logf("[sentinel] Reconciled: transfer DID NOT complete before crash")
		igor.Logf("[sentinel] Compensated — will retry in next cycle")
	}
}

// rng returns a pseudo-random uint64 using xorshift64.
func (s *Sentinel) rng() uint64 {
	s.rngState ^= s.rngState << 13
	s.rngState ^= s.rngState >> 7
	s.rngState ^= s.rngState << 17
	return s.rngState
}

func (s *Sentinel) Marshal() []byte {
	return igor.NewEncoder(256).
		Uint64(s.TickCount).
		Int64(s.BirthNano).
		Int64(s.LastNano).
		Uint64(s.Balance).
		Uint32(s.RefillCount).
		Uint64(s.SpendTotal).
		Bytes(s.Effects.Marshal()).
		Bool(s.Phase != 0).
		Uint64(s.rngState).
		Finish()
}

func (s *Sentinel) Unmarshal(data []byte) {
	d := igor.NewDecoder(data)
	s.TickCount = d.Uint64()
	s.BirthNano = d.Int64()
	s.LastNano = d.Int64()
	s.Balance = d.Uint64()
	s.RefillCount = d.Uint32()
	s.SpendTotal = d.Uint64()
	s.Effects.Unmarshal(d.Bytes()) // THE RESUME RULE: InFlight → Unresolved
	if d.Bool() {
		s.Phase = phaseReconciling
	}
	s.rngState = d.Uint64()
	if err := d.Err(); err != nil {
		panic("unmarshal checkpoint: " + err.Error())
	}
}

func init() { igor.Run(&Sentinel{}) }
func main() {}
