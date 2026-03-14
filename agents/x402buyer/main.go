// SPDX-License-Identifier: Apache-2.0

//go:build tinygo || wasip1

// x402buyer: a demo agent that pays for premium data using the x402 protocol.
//
// The agent demonstrates Igor's payment capabilities:
//   - Fetches data from a paywall endpoint
//   - Receives HTTP 402 with payment terms
//   - Uses wallet_pay to send payment from its budget
//   - Retries with the payment receipt to get premium data
//   - Effect lifecycle ensures crash-safe payments
//
// The paywall server is a mock — no real blockchain involved.
// The point is to demonstrate the pattern: observe → pay → verify → survive.
package main

import (
	"encoding/binary"

	"github.com/simonovic86/igor/sdk/igor"
)

const (
	paywallURL = "http://localhost:8402/api/premium-data"
)

// Phases of the agent.
const (
	phasePolling     uint8 = 0 // normal operation: fetch data
	phaseReconciling uint8 = 1 // handling unresolved intents on resume
)

// X402Buyer implements a demo agent that pays for premium data.
type X402Buyer struct {
	TickCount   uint64
	BirthNano   int64
	LastNano    int64
	Phase       uint8
	PaidCount   uint32 // successful payments
	DataCount   uint32 // premium data received
	TotalPaid   int64  // total microcents paid
	LastReceipt []byte // last payment receipt (for retry)

	// Effect tracking for crash-safe payments.
	Effects igor.EffectLog
}

func (b *X402Buyer) Init() {}

func (b *X402Buyer) Tick() bool {
	b.TickCount++
	now := igor.ClockNow()
	if b.BirthNano == 0 {
		b.BirthNano = now
		igor.Logf("[x402buyer] Agent initialized. Fetching premium data from paywall.")
	}
	b.LastNano = now
	ageSec := (b.LastNano - b.BirthNano) / 1_000_000_000

	// On resume, handle unresolved intents first.
	unresolved := b.Effects.Unresolved()
	if len(unresolved) > 0 {
		b.Phase = phaseReconciling
		for _, intent := range unresolved {
			b.reconcile(intent)
		}
		b.Effects.Prune()
		b.Phase = phasePolling
		return true
	}

	// Check for pending intents that need execution.
	pending := b.Effects.Pending()
	for _, intent := range pending {
		if intent.State == igor.Recorded {
			return b.executePayment(intent)
		}
	}

	// Normal operation: try to fetch premium data.
	// Every 3 ticks, attempt a data fetch (to show repeated payment cycles).
	if b.TickCount%3 != 1 {
		if b.TickCount%5 == 0 {
			igor.Logf("[x402buyer] tick=%d age=%ds payments=%d data=%d total_paid=%d",
				b.TickCount, ageSec, b.PaidCount, b.DataCount, b.TotalPaid)
		}
		return false
	}

	igor.Logf("[x402buyer] Fetching premium data (tick=%d)...", b.TickCount)

	// Try fetching with last receipt if we have one.
	var status int
	var body []byte
	var err error
	if len(b.LastReceipt) > 0 {
		headers := map[string]string{
			"X-Payment": encodeBytesHex(b.LastReceipt),
		}
		status, body, err = igor.HTTPRequest("GET", paywallURL, headers, nil)
	} else {
		status, body, err = igor.HTTPGet(paywallURL)
	}

	if err != nil {
		igor.Logf("[x402buyer] HTTP error: %s", err.Error())
		return false
	}

	if status == 200 {
		// Premium data received!
		b.DataCount++
		b.LastReceipt = nil // Receipt consumed.
		igor.Logf("[x402buyer] PREMIUM DATA RECEIVED: %s", string(body))
		return false
	}

	if status == 402 {
		// Payment required. Parse terms from body.
		amount, recipient, memo := parsePaymentTerms(body)
		if amount <= 0 {
			igor.Logf("[x402buyer] Invalid payment terms in 402 response")
			return false
		}
		igor.Logf("[x402buyer] 402 PAYMENT REQUIRED: %d microcents to %s (%s)",
			amount, recipient, memo)
		return b.recordPaymentIntent(amount, recipient, memo)
	}

	igor.Logf("[x402buyer] Unexpected status %d: %s", status, string(body))
	return false
}

// recordPaymentIntent creates a payment intent. Checkpointed before execution.
func (b *X402Buyer) recordPaymentIntent(amount int64, recipient, memo string) bool {
	var key [16]byte
	_ = igor.RandBytes(key[:])

	data := igor.NewEncoder(128).
		Int64(amount).
		Bytes([]byte(recipient)).
		Bytes([]byte(memo)).
		Finish()

	if err := b.Effects.Record(key[:], data); err != nil {
		igor.Logf("[x402buyer] ERROR: failed to record intent: %s", err.Error())
		return false
	}

	igor.Logf("[x402buyer] Payment intent RECORDED (key=%x...). Waiting for checkpoint...",
		key[:4])
	return false // Wait for checkpoint before execution.
}

// executePayment transitions a Recorded intent to InFlight and sends payment.
func (b *X402Buyer) executePayment(intent igor.Intent) bool {
	// Decode payment parameters.
	d := igor.NewDecoder(intent.Data)
	amount := d.Int64()
	recipient := string(d.Bytes())
	memo := string(d.Bytes())
	if err := d.Err(); err != nil {
		igor.Logf("[x402buyer] ERROR: corrupt intent data: %s", err.Error())
		_ = b.Effects.Compensate(intent.ID)
		return false
	}

	if err := b.Effects.Begin(intent.ID); err != nil {
		igor.Logf("[x402buyer] ERROR: failed to begin intent: %s", err.Error())
		return false
	}

	igor.Logf("[x402buyer] Payment IN-FLIGHT: %d microcents to %s (key=%x...)",
		amount, recipient, intent.ID[:4])

	// === DANGER ZONE: crash between Begin and Confirm → Unresolved ===

	receipt, err := igor.WalletPay(amount, recipient, memo)
	if err != nil {
		igor.Logf("[x402buyer] Payment FAILED: %s", err.Error())
		_ = b.Effects.Compensate(intent.ID)
		b.Effects.Prune()
		return false
	}

	// Payment succeeded.
	b.PaidCount++
	b.TotalPaid += amount
	b.LastReceipt = receipt

	if err := b.Effects.Confirm(intent.ID); err != nil {
		igor.Logf("[x402buyer] ERROR: failed to confirm intent: %s", err.Error())
		return false
	}

	igor.Logf("[x402buyer] Payment CONFIRMED: %d microcents (key=%x..., receipt=%d bytes)",
		amount, intent.ID[:4], len(receipt))
	b.Effects.Prune()
	return true // Fast tick to retry the HTTP request with receipt.
}

// reconcile handles an unresolved payment after crash recovery.
func (b *X402Buyer) reconcile(intent igor.Intent) {
	igor.Logf("[x402buyer] RECONCILING: unresolved payment (key=%x...)", intent.ID[:4])

	d := igor.NewDecoder(intent.Data)
	amount := d.Int64()
	recipient := string(d.Bytes())
	_ = d.Bytes() // memo

	// In production: check if payment was settled on-chain.
	// Here we simulate: if first byte of key is even, payment went through.
	paymentCompleted := (intent.ID[0] % 2) == 0

	if paymentCompleted {
		b.PaidCount++
		b.TotalPaid += amount
		// We don't have the receipt, but the payment is settled.
		if err := b.Effects.Confirm(intent.ID); err != nil {
			igor.Logf("[x402buyer] ERROR: reconcile confirm failed: %s", err.Error())
			return
		}
		igor.Logf("[x402buyer] Reconciled: payment of %d to %s COMPLETED before crash",
			amount, recipient)
	} else {
		if err := b.Effects.Compensate(intent.ID); err != nil {
			igor.Logf("[x402buyer] ERROR: reconcile compensate failed: %s", err.Error())
			return
		}
		igor.Logf("[x402buyer] Reconciled: payment of %d to %s DID NOT complete — will retry",
			amount, recipient)
	}
}

func (b *X402Buyer) Marshal() []byte {
	return igor.NewEncoder(512).
		Uint64(b.TickCount).
		Int64(b.BirthNano).
		Int64(b.LastNano).
		Bool(b.Phase != 0).
		Uint32(b.PaidCount).
		Uint32(b.DataCount).
		Int64(b.TotalPaid).
		Bytes(b.LastReceipt).
		Bytes(b.Effects.Marshal()).
		Finish()
}

func (b *X402Buyer) Unmarshal(data []byte) {
	d := igor.NewDecoder(data)
	b.TickCount = d.Uint64()
	b.BirthNano = d.Int64()
	b.LastNano = d.Int64()
	if d.Bool() {
		b.Phase = phaseReconciling
	}
	b.PaidCount = d.Uint32()
	b.DataCount = d.Uint32()
	b.TotalPaid = d.Int64()
	b.LastReceipt = d.Bytes()
	b.Effects.Unmarshal(d.Bytes()) // THE RESUME RULE: InFlight → Unresolved
	if err := d.Err(); err != nil {
		panic("unmarshal checkpoint: " + err.Error())
	}
}

// parsePaymentTerms extracts amount, recipient, and memo from a 402 response body.
// Expected format (simple binary): [amount:8 LE][recipient_len:4 LE][recipient][memo_len:4 LE][memo]
// Falls back to a fixed default if parsing fails.
func parsePaymentTerms(body []byte) (amount int64, recipient, memo string) {
	if len(body) < 12 {
		return 0, "", ""
	}
	amount = int64(binary.LittleEndian.Uint64(body[:8]))
	off := 8
	recipientLen := int(binary.LittleEndian.Uint32(body[off:]))
	off += 4
	if off+recipientLen > len(body) {
		return 0, "", ""
	}
	recipient = string(body[off : off+recipientLen])
	off += recipientLen
	if off+4 > len(body) {
		return amount, recipient, ""
	}
	memoLen := int(binary.LittleEndian.Uint32(body[off:]))
	off += 4
	if off+memoLen > len(body) {
		return amount, recipient, ""
	}
	memo = string(body[off : off+memoLen])
	return amount, recipient, memo
}

// encodeBytesHex encodes bytes as hex string (simple, no dependencies).
func encodeBytesHex(data []byte) string {
	const hex = "0123456789abcdef"
	buf := make([]byte, len(data)*2)
	for i, b := range data {
		buf[i*2] = hex[b>>4]
		buf[i*2+1] = hex[b&0x0f]
	}
	return string(buf)
}

func init() { igor.Run(&X402Buyer{}) }
func main() {}
