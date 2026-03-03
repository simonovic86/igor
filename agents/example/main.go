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

func (s *Survivor) Tick() {
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
}

func (s *Survivor) Marshal() []byte {
	buf := make([]byte, stateSize)
	binary.LittleEndian.PutUint64(buf[0:8], s.TickCount)
	binary.LittleEndian.PutUint64(buf[8:16], uint64(s.BirthNano))
	binary.LittleEndian.PutUint64(buf[16:24], uint64(s.LastNano))
	binary.LittleEndian.PutUint32(buf[24:28], s.Luck)
	return buf
}

func (s *Survivor) Unmarshal(data []byte) {
	if len(data) < stateSize {
		return
	}
	s.TickCount = binary.LittleEndian.Uint64(data[0:8])
	s.BirthNano = int64(binary.LittleEndian.Uint64(data[8:16]))
	s.LastNano = int64(binary.LittleEndian.Uint64(data[16:24]))
	s.Luck = binary.LittleEndian.Uint32(data[24:28])
}

func init() { igor.Run(&Survivor{}) }
func main() {}
