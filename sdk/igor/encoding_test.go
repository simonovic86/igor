package igor

import (
	"bytes"
	"testing"
)

func TestEncoder_RoundTrip(t *testing.T) {
	encoded := NewEncoder(64).
		Uint64(42).
		Int64(-100).
		Uint32(999).
		Int32(-1).
		Bool(true).
		Bool(false).
		Bytes([]byte{0xDE, 0xAD}).
		String("hello").
		Finish()

	d := NewDecoder(encoded)
	if v := d.Uint64(); v != 42 {
		t.Errorf("Uint64: got %d, want 42", v)
	}
	if v := d.Int64(); v != -100 {
		t.Errorf("Int64: got %d, want -100", v)
	}
	if v := d.Uint32(); v != 999 {
		t.Errorf("Uint32: got %d, want 999", v)
	}
	if v := d.Int32(); v != -1 {
		t.Errorf("Int32: got %d, want -1", v)
	}
	if v := d.Bool(); v != true {
		t.Errorf("Bool: got %v, want true", v)
	}
	if v := d.Bool(); v != false {
		t.Errorf("Bool: got %v, want false", v)
	}
	if v := d.Bytes(); !bytes.Equal(v, []byte{0xDE, 0xAD}) {
		t.Errorf("Bytes: got %v, want [0xDE 0xAD]", v)
	}
	if v := d.String(); v != "hello" {
		t.Errorf("String: got %q, want %q", v, "hello")
	}
	if err := d.Err(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDecoder_ShortRead(t *testing.T) {
	// Try to read a Uint64 from 4 bytes
	d := NewDecoder([]byte{0x01, 0x02, 0x03, 0x04})
	v := d.Uint64()
	if v != 0 {
		t.Errorf("expected 0 on short read, got %d", v)
	}
	if d.Err() == nil {
		t.Error("expected error on short read")
	}

	// Subsequent reads should also fail
	v32 := d.Uint32()
	if v32 != 0 {
		t.Errorf("expected 0 after error, got %d", v32)
	}
}

func TestDecoder_ShortRead_Bytes(t *testing.T) {
	// Encode a 10-byte slice length, but only provide 5 bytes of data
	encoded := NewEncoder(14).Uint32(10).Finish()
	encoded = append(encoded, 0x01, 0x02, 0x03, 0x04, 0x05) // only 5 of 10

	d := NewDecoder(encoded)
	b := d.Bytes()
	if b != nil {
		t.Errorf("expected nil on short bytes, got %v", b)
	}
	if d.Err() == nil {
		t.Error("expected error on short bytes read")
	}
}

func TestEncoder_EmptyBytes(t *testing.T) {
	encoded := NewEncoder(4).Bytes(nil).Finish()

	d := NewDecoder(encoded)
	b := d.Bytes()
	if len(b) != 0 {
		t.Errorf("expected empty bytes, got %v", b)
	}
	if d.Err() != nil {
		t.Errorf("unexpected error: %v", d.Err())
	}
}

func TestEncoder_EmptyString(t *testing.T) {
	encoded := NewEncoder(4).String("").Finish()

	d := NewDecoder(encoded)
	s := d.String()
	if s != "" {
		t.Errorf("expected empty string, got %q", s)
	}
	if d.Err() != nil {
		t.Errorf("unexpected error: %v", d.Err())
	}
}

func TestDecoder_Bool_ShortRead(t *testing.T) {
	d := NewDecoder([]byte{})
	v := d.Bool()
	if v != false {
		t.Errorf("expected false on short read, got %v", v)
	}
	if d.Err() == nil {
		t.Error("expected error on empty bool read")
	}
}

func TestEncoder_SurvivorPattern(t *testing.T) {
	// Simulate the Survivor agent's Marshal/Unmarshal pattern
	type Survivor struct {
		TickCount uint64
		BirthNano int64
		LastNano  int64
		Luck      uint32
	}

	original := Survivor{
		TickCount: 42,
		BirthNano: 1000000000,
		LastNano:  5000000000,
		Luck:      0xDEADBEEF,
	}

	encoded := NewEncoder(28).
		Uint64(original.TickCount).
		Int64(original.BirthNano).
		Int64(original.LastNano).
		Uint32(original.Luck).
		Finish()

	if len(encoded) != 28 {
		t.Fatalf("expected 28 bytes, got %d", len(encoded))
	}

	d := NewDecoder(encoded)
	restored := Survivor{
		TickCount: d.Uint64(),
		BirthNano: d.Int64(),
		LastNano:  d.Int64(),
		Luck:      d.Uint32(),
	}

	if d.Err() != nil {
		t.Fatalf("decode error: %v", d.Err())
	}
	if restored != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", restored, original)
	}
}
