// SPDX-License-Identifier: Apache-2.0

//go:build tinygo || wasip1

package main

import (
	"github.com/simonovic86/igor/sdk/igor"
)

// stateSize is the fixed checkpoint size in bytes:
// TickCount(8) + BirthNano(8) + LastNano(8) +
// BTCLatest(8) + BTCHigh(8) + BTCLow(8) +
// ETHLatest(8) + ETHHigh(8) + ETHLow(8) +
// ObservationCount(4) + ErrorCount(4)
const stateSize = 80

const apiURL = "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin,ethereum&vs_currencies=usd"

// fetchInterval is how many ticks between API calls (CoinGecko free tier: ~30 req/min).
const fetchInterval = 10

// PriceWatcher fetches BTC and ETH prices from CoinGecko on every tick,
// tracks all-time high/low, and logs price updates. All state survives
// checkpoint/resume across machines — proving portable, immortal agents
// that accumulate real-world knowledge over time.
type PriceWatcher struct {
	TickCount        uint64
	BirthNano        int64
	LastNano         int64
	BTCLatest        uint64 // price in millicents (USD × 100)
	BTCHigh          uint64
	BTCLow           uint64
	ETHLatest        uint64
	ETHHigh          uint64
	ETHLow           uint64
	ObservationCount uint32
	ErrorCount       uint32
}

func (p *PriceWatcher) Init() {}

func (p *PriceWatcher) Tick() bool {
	p.TickCount++

	now := igor.ClockNow()
	if p.BirthNano == 0 {
		p.BirthNano = now
	}
	p.LastNano = now

	ageSec := (p.LastNano - p.BirthNano) / 1_000_000_000

	// Fetch prices on tick 1 and every fetchInterval ticks after that.
	if p.TickCount == 1 || p.TickCount%fetchInterval == 0 {
		status, body, err := igor.HTTPGet(apiURL)
		if err != nil {
			p.ErrorCount++
			igor.Logf("[pricewatcher] tick=%d ERROR: http failed: %s (errors=%d)",
				p.TickCount, err.Error(), p.ErrorCount)
			return false
		}
		if status < 200 || status >= 300 {
			p.ErrorCount++
			igor.Logf("[pricewatcher] tick=%d ERROR: http status %d (errors=%d)",
				p.TickCount, status, p.ErrorCount)
			return false
		}

		btc := extractPrice(string(body), "bitcoin")
		eth := extractPrice(string(body), "ethereum")

		if btc == 0 && eth == 0 {
			p.ErrorCount++
			igor.Logf("[pricewatcher] tick=%d ERROR: could not parse prices (errors=%d)",
				p.TickCount, p.ErrorCount)
			return false
		}

		p.ObservationCount++

		if btc > 0 {
			p.BTCLatest = btc
			if p.BTCHigh == 0 || btc > p.BTCHigh {
				p.BTCHigh = btc
			}
			if p.BTCLow == 0 || btc < p.BTCLow {
				p.BTCLow = btc
			}
		}

		if eth > 0 {
			p.ETHLatest = eth
			if p.ETHHigh == 0 || eth > p.ETHHigh {
				p.ETHHigh = eth
			}
			if p.ETHLow == 0 || eth < p.ETHLow {
				p.ETHLow = eth
			}
		}

		igor.Logf("[pricewatcher] tick=%d age=%ds obs=%d FETCH | BTC=$%d.%02d (high=$%d.%02d low=$%d.%02d) | ETH=$%d.%02d (high=$%d.%02d low=$%d.%02d)",
			p.TickCount, ageSec, p.ObservationCount,
			p.BTCLatest/100, p.BTCLatest%100, p.BTCHigh/100, p.BTCHigh%100, p.BTCLow/100, p.BTCLow%100,
			p.ETHLatest/100, p.ETHLatest%100, p.ETHHigh/100, p.ETHHigh%100, p.ETHLow/100, p.ETHLow%100)
	} else if p.TickCount%5 == 0 {
		// Log a heartbeat every 5 ticks with cached prices.
		igor.Logf("[pricewatcher] tick=%d age=%ds obs=%d | BTC=$%d.%02d | ETH=$%d.%02d",
			p.TickCount, ageSec, p.ObservationCount,
			p.BTCLatest/100, p.BTCLatest%100,
			p.ETHLatest/100, p.ETHLatest%100)
	}

	if p.TickCount%10 == 0 {
		igor.Logf("[pricewatcher] MILESTONE: %d observations across %ds — this agent remembers everything",
			p.ObservationCount, ageSec)
	}

	return false
}

func (p *PriceWatcher) Marshal() []byte {
	return igor.NewEncoder(stateSize).
		Uint64(p.TickCount).
		Int64(p.BirthNano).
		Int64(p.LastNano).
		Uint64(p.BTCLatest).
		Uint64(p.BTCHigh).
		Uint64(p.BTCLow).
		Uint64(p.ETHLatest).
		Uint64(p.ETHHigh).
		Uint64(p.ETHLow).
		Uint32(p.ObservationCount).
		Uint32(p.ErrorCount).
		Finish()
}

func (p *PriceWatcher) Unmarshal(data []byte) {
	d := igor.NewDecoder(data)
	p.TickCount = d.Uint64()
	p.BirthNano = d.Int64()
	p.LastNano = d.Int64()
	p.BTCLatest = d.Uint64()
	p.BTCHigh = d.Uint64()
	p.BTCLow = d.Uint64()
	p.ETHLatest = d.Uint64()
	p.ETHHigh = d.Uint64()
	p.ETHLow = d.Uint64()
	p.ObservationCount = d.Uint32()
	p.ErrorCount = d.Uint32()
	if err := d.Err(); err != nil {
		panic("unmarshal checkpoint: " + err.Error())
	}
}

// extractPrice parses a CoinGecko simple/price JSON response to find
// the USD price for a given coin. Returns price in cents (USD × 100).
// Uses simple string scanning — no encoding/json needed (TinyGo-safe).
//
// Expected format: {"bitcoin":{"usd":83456.78},"ethereum":{"usd":1923.45}}
func extractPrice(body, coin string) uint64 {
	// Find "bitcoin":{"usd": or "ethereum":{"usd":
	key := `"` + coin + `":{"usd":`
	idx := indexOf(body, key)
	if idx < 0 {
		return 0
	}

	// Skip past the key to the number.
	start := idx + len(key)
	if start >= len(body) {
		return 0
	}

	// Parse the number (integer part and optional decimal).
	var whole uint64
	var frac uint64
	var fracDigits int
	inFrac := false
	i := start

	// Skip whitespace.
	for i < len(body) && body[i] == ' ' {
		i++
	}

	for i < len(body) {
		c := body[i]
		if c >= '0' && c <= '9' {
			if inFrac {
				if fracDigits < 2 {
					frac = frac*10 + uint64(c-'0')
					fracDigits++
				}
				// Skip additional decimal digits.
			} else {
				whole = whole*10 + uint64(c-'0')
			}
		} else if c == '.' && !inFrac {
			inFrac = true
		} else {
			break
		}
		i++
	}

	// Pad fractional part to 2 digits.
	for fracDigits < 2 {
		frac *= 10
		fracDigits++
	}

	return whole*100 + frac
}

// indexOf returns the index of the first occurrence of needle in s, or -1.
func indexOf(s, needle string) int {
	if len(needle) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(needle); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if s[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func init() { igor.Run(&PriceWatcher{}) }
func main() {}
