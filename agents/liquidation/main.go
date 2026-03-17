// SPDX-License-Identifier: Apache-2.0

//go:build tinygo || wasip1

package main

import (
	"github.com/simonovic86/igor/sdk/igor"
)

// stateSize is the fixed checkpoint size in bytes:
// TickCount(8) + BirthNano(8) + LastNano(8) + LastProcessedSlot(8) +
// PriceLatest(8) + PriceHigh(8) + PriceLow(8) + FirstWarningSlot(8) +
// WarningActive(4) + SlotsProcessed(4)
const stateSize = 72

// Liquidation threshold in cents. Position: 10 ETH collateral, 15,000 USDC debt.
// Liquidation when collateral value ($ETH_price × 10) approaches debt.
// Threshold: ETH = $1,550.00 → collateral = $15,500 (just above $15,000 debt).
const liquidationThresholdCents = 155000

// LiquidationWatcher tracks a simulated ETH position against a deterministic
// price curve. On resume after downtime, it detects the gap, replays missed
// time slots, and discovers events that occurred while it was absent.
//
// The world is a pure function of time — it exists whether or not the agent
// is running. The agent's job is to stay continuous with that world.
type LiquidationWatcher struct {
	TickCount         uint64
	BirthNano         int64
	LastNano          int64
	LastProcessedSlot uint64
	PriceLatest       uint64 // cents (e.g., 200000 = $2,000.00)
	PriceHigh         uint64
	PriceLow          uint64
	FirstWarningSlot  uint64 // first slot threshold breached (0 = never)
	WarningActive     uint32 // 1 = currently below threshold
	SlotsProcessed    uint32
}

// Price curve control points: slot → price in cents.
// Authored for narrative tension, not randomness.
//
//	 0–15:  Calm with mild volatility. Agent is alive.
//	16–30:  Increasing pressure. Agent will die around slot 15.
//	31–45:  Sharp drop, threshold breach at slot 38, partial recovery.
//	        Agent is dead for all of this.
//	46–60:  Recovery and stabilization. Agent resumes here.
type controlPoint struct {
	slot  uint64
	price uint64 // cents
}

var curve = []controlPoint{
	{0, 200000},  // $2,000.00 — opening
	{5, 205000},  // $2,050.00 — mild upward drift
	{10, 192000}, // $1,920.00 — first dip
	{15, 198000}, // $1,980.00 — partial recovery
	{20, 185000}, // $1,850.00 — renewed pressure
	{25, 178000}, // $1,780.00 — drawdown deepens
	{30, 182000}, // $1,820.00 — small bounce (last checkpoint before crash)
	{35, 165000}, // $1,650.00 — sharp drop during outage
	{38, 154000}, // $1,540.00 — THRESHOLD BREACHED during outage
	{40, 158000}, // $1,580.00 — partial recovery, still danger zone
	{45, 162000}, // $1,620.00 — further recovery
	{50, 170000}, // $1,700.00 — stabilizing
	{55, 175000}, // $1,750.00 — continued recovery
}

// priceAtSlot returns the ETH price in cents for a given time slot.
// Linear interpolation between control points; hold last value beyond curve.
func priceAtSlot(slot uint64) uint64 {
	if slot <= curve[0].slot {
		return curve[0].price
	}
	for i := 1; i < len(curve); i++ {
		if slot <= curve[i].slot {
			prev := curve[i-1]
			next := curve[i]
			span := next.slot - prev.slot
			offset := slot - prev.slot
			if next.price >= prev.price {
				return prev.price + (next.price-prev.price)*offset/span
			}
			return prev.price - (prev.price-next.price)*offset/span
		}
	}
	// Beyond last control point: hold last price.
	return curve[len(curve)-1].price
}

func (w *LiquidationWatcher) Init() {}

func (w *LiquidationWatcher) Tick() bool {
	w.TickCount++

	now := igor.ClockNow()
	if w.BirthNano == 0 {
		w.BirthNano = now
	}
	w.LastNano = now

	currentSlot := uint64((now - w.BirthNano) / 1_000_000_000)

	// First tick ever: initialize and process slot 0.
	if w.TickCount == 1 {
		w.processSlot(0, "live")
		w.LastProcessedSlot = 0
		return false
	}

	// Detect gap.
	if currentSlot > w.LastProcessedSlot+1 {
		gap := currentSlot - w.LastProcessedSlot - 1
		igor.Logf("")
		igor.Logf("────────────────────────────────────────────────────")
		igor.Logf("  GAP DETECTED")
		igor.Logf("  Last processed:  slot %d", w.LastProcessedSlot)
		igor.Logf("  Current time:    slot %d", currentSlot)
		igor.Logf("  Missed:          %d slots (%d..%d)",
			gap, w.LastProcessedSlot+1, currentSlot-1)
		igor.Logf("  Replaying missed history...")
		igor.Logf("────────────────────────────────────────────────────")

		// Remember whether warning was already known before catch-up.
		warningBefore := w.FirstWarningSlot

		// Catch up on missed slots.
		for slot := w.LastProcessedSlot + 1; slot < currentSlot; slot++ {
			w.processSlot(slot, "catch-up")
		}

		// Summary after catch-up.
		igor.Logf("────────────────────────────────────────────────────")
		if w.FirstWarningSlot != 0 && warningBefore == 0 {
			igor.Logf("  ⚠ THRESHOLD BREACHED AT SLOT %d (DURING DOWNTIME)", w.FirstWarningSlot)
			igor.Logf("  The agent was dead. The world kept moving.")
			igor.Logf("  Catch-up reconstructed %d missed slots.", gap)
		} else {
			igor.Logf("  Catch-up complete: %d missed slots replayed.", gap)
		}
		igor.Logf("────────────────────────────────────────────────────")
		igor.Logf("")
	}

	// Process current slot as live.
	w.processSlot(currentSlot, "live")
	w.LastProcessedSlot = currentSlot

	return false
}

func (w *LiquidationWatcher) processSlot(slot uint64, mode string) {
	price := priceAtSlot(slot)

	w.PriceLatest = price
	if w.PriceHigh == 0 || price > w.PriceHigh {
		w.PriceHigh = price
	}
	if w.PriceLow == 0 || price < w.PriceLow {
		w.PriceLow = price
	}
	w.SlotsProcessed++

	belowThreshold := price < liquidationThresholdCents

	if belowThreshold && w.FirstWarningSlot == 0 {
		w.FirstWarningSlot = slot
	}
	if belowThreshold {
		w.WarningActive = 1
	} else {
		w.WarningActive = 0
	}

	// Compute distance to threshold.
	var distSign string
	var dist uint64
	if price >= liquidationThresholdCents {
		dist = price - liquidationThresholdCents
		distSign = "+"
	} else {
		dist = liquidationThresholdCents - price
		distSign = "-"
	}

	if belowThreshold {
		if slot == w.FirstWarningSlot {
			igor.Logf("[%s] slot %d: ETH $%d.%02d  threshold $%d.%02d  distance %s$%d.%02d  ⚠ LIQUIDATION WARNING (threshold breached)",
				mode, slot,
				price/100, price%100,
				liquidationThresholdCents/100, liquidationThresholdCents%100,
				distSign, dist/100, dist%100)
		} else {
			igor.Logf("[%s] slot %d: ETH $%d.%02d  threshold $%d.%02d  distance %s$%d.%02d  ⚠ WARNING",
				mode, slot,
				price/100, price%100,
				liquidationThresholdCents/100, liquidationThresholdCents%100,
				distSign, dist/100, dist%100)
		}
	} else {
		igor.Logf("[%s] slot %d: ETH $%d.%02d  threshold $%d.%02d  distance %s$%d.%02d",
			mode, slot,
			price/100, price%100,
			liquidationThresholdCents/100, liquidationThresholdCents%100,
			distSign, dist/100, dist%100)
	}
}

func (w *LiquidationWatcher) Marshal() []byte {
	return igor.NewEncoder(stateSize).
		Uint64(w.TickCount).
		Int64(w.BirthNano).
		Int64(w.LastNano).
		Uint64(w.LastProcessedSlot).
		Uint64(w.PriceLatest).
		Uint64(w.PriceHigh).
		Uint64(w.PriceLow).
		Uint64(w.FirstWarningSlot).
		Uint32(w.WarningActive).
		Uint32(w.SlotsProcessed).
		Finish()
}

func (w *LiquidationWatcher) Unmarshal(data []byte) {
	d := igor.NewDecoder(data)
	w.TickCount = d.Uint64()
	w.BirthNano = d.Int64()
	w.LastNano = d.Int64()
	w.LastProcessedSlot = d.Uint64()
	w.PriceLatest = d.Uint64()
	w.PriceHigh = d.Uint64()
	w.PriceLow = d.Uint64()
	w.FirstWarningSlot = d.Uint64()
	w.WarningActive = d.Uint32()
	w.SlotsProcessed = d.Uint32()
	if err := d.Err(); err != nil {
		panic("unmarshal checkpoint: " + err.Error())
	}
}

func init() { igor.Run(&LiquidationWatcher{}) }
func main() {}
