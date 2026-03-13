// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"crypto/ed25519"
	"strings"
	"testing"
)

func TestDID_Format(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	did := id.DID()
	if !strings.HasPrefix(did, "did:key:z6Mk") {
		t.Fatalf("DID should start with 'did:key:z6Mk', got: %s", did)
	}
}

func TestDID_RoundTrip(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	did := id.DID()
	pubKey, err := ParseDID(did)
	if err != nil {
		t.Fatalf("ParseDID() error: %v", err)
	}
	if !ed25519.PublicKey(pubKey).Equal(id.PublicKey) {
		t.Fatal("public key mismatch after DID round-trip")
	}
}

func TestDID_Deterministic(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	did1 := id.DID()
	did2 := id.DID()
	if did1 != did2 {
		t.Fatalf("DID not deterministic: %s != %s", did1, did2)
	}
}

func TestDID_UniquePerIdentity(t *testing.T) {
	id1, _ := Generate()
	id2, _ := Generate()
	if id1.DID() == id2.DID() {
		t.Fatal("two different identities produced the same DID")
	}
}

func TestDIDShort(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	short := id.DIDShort()
	if !strings.HasPrefix(short, "did:key:z6Mk") {
		t.Fatalf("DIDShort should start with 'did:key:z6Mk', got: %s", short)
	}
	if !strings.Contains(short, "...") {
		t.Fatalf("DIDShort should contain '...', got: %s", short)
	}
	if len(short) > 24 {
		t.Fatalf("DIDShort too long: %d chars: %s", len(short), short)
	}
}

func TestParseDID_InvalidPrefix(t *testing.T) {
	_, err := ParseDID("did:web:example.com")
	if err == nil {
		t.Fatal("expected error for non-did:key DID")
	}
}

func TestParseDID_TooShort(t *testing.T) {
	_, err := ParseDID("did:key:z")
	if err == nil {
		t.Fatal("expected error for too-short DID")
	}
}

func TestParseDID_InvalidBase58(t *testing.T) {
	_, err := ParseDID("did:key:z0OOO") // 0 and O are not in base58
	if err == nil {
		t.Fatal("expected error for invalid base58 characters")
	}
}

func TestBase58_RoundTrip(t *testing.T) {
	testCases := [][]byte{
		{0x00},
		{0x00, 0x00, 0x01},
		{0xed, 0x01, 0xff, 0xaa, 0xbb},
		make([]byte, 32),
	}
	for _, data := range testCases {
		encoded := base58Encode(data)
		decoded, err := base58Decode(encoded)
		if err != nil {
			t.Fatalf("base58Decode(%q) error: %v", encoded, err)
		}
		if len(decoded) != len(data) {
			t.Fatalf("length mismatch: got %d, want %d", len(decoded), len(data))
		}
		for i := range data {
			if decoded[i] != data[i] {
				t.Fatalf("byte %d mismatch: got 0x%02x, want 0x%02x", i, decoded[i], data[i])
			}
		}
	}
}
