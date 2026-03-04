//go:build tinygo || wasip1

package main

import (
	"encoding/binary"

	"github.com/simonovic86/igor/sdk/igor"
)

const stateSize = 28

type Survivor struct {
	TickCount uint64
	BirthNano int64
	LastNano  int64
	Luck      uint32
}

func (s *Survivor) Init() {}

func (s *Survivor) Tick() bool {
	s.TickCount++

	now := igor.ClockNow()
	if s.BirthNano == 0 {
		s.BirthNano = now
	}
	s.LastNano = now

	randBuf := make([]byte, 4)
	_ = igor.RandBytes(randBuf)
	s.Luck ^= binary.LittleEndian.Uint32(randBuf)

	ageSec := (s.LastNano - s.BirthNano) / 1_000_000_000
	igor.Logf("[survivor] tick %d | age %ds | luck 0x%08x",
		s.TickCount, ageSec, s.Luck)

	if s.TickCount%10 == 0 {
		igor.Logf("[survivor] milestone: survived %d ticks", s.TickCount)
	}
	return false
}

func (s *Survivor) Marshal() []byte {
	return igor.NewEncoder(stateSize).
		Uint64(s.TickCount).
		Int64(s.BirthNano).
		Int64(s.LastNano).
		Uint32(s.Luck).
		Finish()
}

func (s *Survivor) Unmarshal(data []byte) {
	d := igor.NewDecoder(data)
	s.TickCount = d.Uint64()
	s.BirthNano = d.Int64()
	s.LastNano = d.Int64()
	s.Luck = d.Uint32()
}

func init() { igor.Run(&Survivor{}) }
func main() {}
