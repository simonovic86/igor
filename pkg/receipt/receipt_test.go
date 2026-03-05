// SPDX-License-Identifier: Apache-2.0

package receipt

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func testReceipt() Receipt {
	return Receipt{
		AgentID:        "test-agent",
		NodeID:         "12D3KooWAbCdEfGhIjKlMnOpQrStUvWxYz",
		EpochStart:     1,
		EpochEnd:       5,
		CostMicrocents: 5000,
		BudgetAfter:    9995000,
		Timestamp:      1700000000000000000,
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	r := testReceipt()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	if err := r.Sign(priv); err != nil {
		t.Fatal(err)
	}

	data := r.MarshalBinary()
	got, err := UnmarshalBinary(data)
	if err != nil {
		t.Fatal(err)
	}

	if got.AgentID != r.AgentID {
		t.Errorf("AgentID: got %q, want %q", got.AgentID, r.AgentID)
	}
	if got.NodeID != r.NodeID {
		t.Errorf("NodeID: got %q, want %q", got.NodeID, r.NodeID)
	}
	if got.EpochStart != r.EpochStart {
		t.Errorf("EpochStart: got %d, want %d", got.EpochStart, r.EpochStart)
	}
	if got.EpochEnd != r.EpochEnd {
		t.Errorf("EpochEnd: got %d, want %d", got.EpochEnd, r.EpochEnd)
	}
	if got.CostMicrocents != r.CostMicrocents {
		t.Errorf("CostMicrocents: got %d, want %d", got.CostMicrocents, r.CostMicrocents)
	}
	if got.BudgetAfter != r.BudgetAfter {
		t.Errorf("BudgetAfter: got %d, want %d", got.BudgetAfter, r.BudgetAfter)
	}
	if got.Timestamp != r.Timestamp {
		t.Errorf("Timestamp: got %d, want %d", got.Timestamp, r.Timestamp)
	}

	if !got.Verify(pub) {
		t.Error("signature verification failed after round-trip")
	}
}

func TestSignVerify(t *testing.T) {
	r := testReceipt()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	if err := r.Sign(priv); err != nil {
		t.Fatal(err)
	}

	if !r.Verify(pub) {
		t.Error("valid signature not accepted")
	}
}

func TestSignVerify_WrongKey(t *testing.T) {
	r := testReceipt()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub2, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	if err := r.Sign(priv); err != nil {
		t.Fatal(err)
	}

	if r.Verify(pub2) {
		t.Error("wrong key accepted")
	}
}

func TestSignVerify_TamperedData(t *testing.T) {
	r := testReceipt()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	if err := r.Sign(priv); err != nil {
		t.Fatal(err)
	}

	// Tamper with cost
	r.CostMicrocents = 9999
	if r.Verify(pub) {
		t.Error("tampered receipt accepted")
	}
}

func TestMarshalReceipts_Multiple(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	receipts := make([]Receipt, 3)
	for i := range receipts {
		receipts[i] = testReceipt()
		receipts[i].EpochStart = uint64(i*5 + 1)
		receipts[i].EpochEnd = uint64((i + 1) * 5)
		if err := receipts[i].Sign(priv); err != nil {
			t.Fatal(err)
		}
	}

	data := MarshalReceipts(receipts)
	got, err := UnmarshalReceipts(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != len(receipts) {
		t.Fatalf("count: got %d, want %d", len(got), len(receipts))
	}

	for i := range got {
		if got[i].EpochStart != receipts[i].EpochStart {
			t.Errorf("receipt %d EpochStart: got %d, want %d", i, got[i].EpochStart, receipts[i].EpochStart)
		}
		if got[i].EpochEnd != receipts[i].EpochEnd {
			t.Errorf("receipt %d EpochEnd: got %d, want %d", i, got[i].EpochEnd, receipts[i].EpochEnd)
		}
	}
}

func TestMarshalReceipts_Empty(t *testing.T) {
	data := MarshalReceipts(nil)
	got, err := UnmarshalReceipts(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d receipts", len(got))
	}
}

func TestUnmarshalBinary_TooShort(t *testing.T) {
	_, err := UnmarshalBinary([]byte{1, 2})
	if err == nil {
		t.Error("expected error for short data")
	}
}

func TestUnmarshalReceipts_TooShort(t *testing.T) {
	_, err := UnmarshalReceipts([]byte{1})
	if err == nil {
		t.Error("expected error for short data")
	}
}

func TestVerify_BadKeySize(t *testing.T) {
	r := testReceipt()
	if r.Verify([]byte{1, 2, 3}) {
		t.Error("bad key size should fail verification")
	}
}

func TestSign_BadKeySize(t *testing.T) {
	r := testReceipt()
	if err := r.Sign([]byte{1, 2, 3}); err == nil {
		t.Error("bad key size should fail signing")
	}
}

func TestUnmarshalReceipts_TruncatedEntryLength(t *testing.T) {
	// Valid count header (count=2) but only enough bytes for the count header
	// and the first receipt's length prefix — no actual receipt data.
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	r := testReceipt()
	if err := r.Sign(priv); err != nil {
		t.Fatal(err)
	}
	rb := r.MarshalBinary()

	// Build: [count=2][len1][receipt1] — missing second receipt entirely
	buf := make([]byte, 0, 4+4+len(rb))
	buf = append(buf, 0, 0, 0, 0)               // count placeholder
	buf[0], buf[1], buf[2], buf[3] = 2, 0, 0, 0 // count = 2
	lenBytes := make([]byte, 4)
	lenBytes[0] = byte(len(rb))
	lenBytes[1] = byte(len(rb) >> 8)
	lenBytes[2] = byte(len(rb) >> 16)
	lenBytes[3] = byte(len(rb) >> 24)
	buf = append(buf, lenBytes...)
	buf = append(buf, rb...)
	// No second receipt — should fail at "entry 1 length"

	_, err = UnmarshalReceipts(buf)
	if err == nil {
		t.Error("expected error for truncated receipts (missing second entry)")
	}
}

func TestUnmarshalReceipts_TruncatedEntryData(t *testing.T) {
	// count=1, length=100, but only 10 bytes of data
	buf := make([]byte, 0, 4+4+10)
	buf = append(buf, 1, 0, 0, 0)          // count = 1
	buf = append(buf, 100, 0, 0, 0)        // length = 100
	buf = append(buf, make([]byte, 10)...) // only 10 bytes, not 100

	_, err := UnmarshalReceipts(buf)
	if err == nil {
		t.Error("expected error for truncated receipt data")
	}
}

func TestUnmarshalBinary_TruncatedSignature(t *testing.T) {
	// Build a receipt with a sig_len that claims 64 bytes but only has 2
	r := testReceipt()
	signData := r.MarshalSignData()

	buf := make([]byte, 0, len(signData)+4+2)
	buf = append(buf, signData...)
	buf = append(buf, 64, 0, 0, 0) // sig_len = 64
	buf = append(buf, 0xAA, 0xBB)  // only 2 bytes, not 64

	_, err := UnmarshalBinary(buf)
	if err == nil {
		t.Error("expected error for truncated signature")
	}
}

func TestUnmarshalBinary_TruncatedFields(t *testing.T) {
	// Build valid sign data but truncate in the middle of a uint64 field
	r := testReceipt()
	full := r.MarshalSignData()

	// Truncate after agentID + nodeID strings, in the middle of epochStart
	// agentID: 4 + 10 = 14 bytes, nodeID: 4 + 36 = 40 bytes, total = 54 bytes
	// epochStart needs 8 more bytes — truncate at 58 (4 bytes into epochStart)
	truncated := full[:58]
	_, err := UnmarshalBinary(truncated)
	if err == nil {
		t.Error("expected error for truncated receipt fields")
	}
}
