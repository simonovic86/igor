// SPDX-License-Identifier: Apache-2.0

//go:build tinygo || wasip1

package main

import (
	"github.com/simonovic86/igor/sdk/igor"
)

const stateSize = 28

// Heartbeat is a demo agent that emits visible heartbeat logs on every tick.
// It demonstrates continuity across checkpoint/resume/migration by tracking
// tick count and birth time — both survive transparent migration.
type Heartbeat struct {
	TickCount  uint64
	BirthNano  int64
	LastNano   int64
	MessageNum uint32
}

func (h *Heartbeat) Init() {}

func (h *Heartbeat) Tick() bool {
	h.TickCount++

	now := igor.ClockNow()
	if h.BirthNano == 0 {
		h.BirthNano = now
	}
	h.LastNano = now

	ageSec := (h.LastNano - h.BirthNano) / 1_000_000_000

	igor.Logf("[heartbeat] tick=%d age=%ds", h.TickCount, ageSec)

	if h.TickCount%10 == 0 {
		h.MessageNum++
		igor.Logf("[heartbeat] MILESTONE #%d: survived %d ticks across %ds",
			h.MessageNum, h.TickCount, ageSec)
	}

	return false
}

func (h *Heartbeat) Marshal() []byte {
	return igor.NewEncoder(stateSize).
		Uint64(h.TickCount).
		Int64(h.BirthNano).
		Int64(h.LastNano).
		Uint32(h.MessageNum).
		Finish()
}

func (h *Heartbeat) Unmarshal(data []byte) {
	d := igor.NewDecoder(data)
	h.TickCount = d.Uint64()
	h.BirthNano = d.Int64()
	h.LastNano = d.Int64()
	h.MessageNum = d.Uint32()
	if err := d.Err(); err != nil {
		panic("unmarshal checkpoint: " + err.Error())
	}
}

func init() { igor.Run(&Heartbeat{}) }
func main() {}
