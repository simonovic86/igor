// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestGenerate(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if len(id.PublicKey) != ed25519.PublicKeySize {
		t.Fatalf("public key size = %d, want %d", len(id.PublicKey), ed25519.PublicKeySize)
	}
	if len(id.PrivateKey) != ed25519.PrivateKeySize {
		t.Fatalf("private key size = %d, want %d", len(id.PrivateKey), ed25519.PrivateKeySize)
	}
	// Verify the keypair works for signing
	msg := []byte("test message")
	sig := ed25519.Sign(id.PrivateKey, msg)
	if !ed25519.Verify(id.PublicKey, msg, sig) {
		t.Fatal("generated keypair failed sign/verify round-trip")
	}
}

func TestGenerate_UniqueKeys(t *testing.T) {
	id1, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	id2, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if id1.PublicKey.Equal(id2.PublicKey) {
		t.Fatal("two generated identities have the same public key")
	}
}

func TestFromPrivateKey(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey error: %v", err)
	}
	id, err := FromPrivateKey(priv)
	if err != nil {
		t.Fatalf("FromPrivateKey error: %v", err)
	}
	expectedPub := priv.Public().(ed25519.PublicKey)
	if !id.PublicKey.Equal(expectedPub) {
		t.Fatal("public key mismatch after FromPrivateKey")
	}
}

func TestFromPrivateKey_BadSize(t *testing.T) {
	_, err := FromPrivateKey(ed25519.PrivateKey(make([]byte, 10)))
	if err == nil {
		t.Fatal("expected error for bad private key size")
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	original, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	data := original.MarshalBinary()
	restored, err := UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("UnmarshalBinary error: %v", err)
	}
	if !original.PublicKey.Equal(restored.PublicKey) {
		t.Fatal("public key mismatch after round-trip")
	}
	if !original.PrivateKey.Equal(restored.PrivateKey) {
		t.Fatal("private key mismatch after round-trip")
	}
}

func TestUnmarshalBinary_TooShort(t *testing.T) {
	_, err := UnmarshalBinary([]byte{0x01, 0x02})
	if err == nil {
		t.Fatal("expected error for too-short data")
	}
}

func TestUnmarshalBinary_BadKeyLen(t *testing.T) {
	// Encode a key length of 99 (invalid)
	data := make([]byte, 4+99)
	data[0] = 99
	_, err := UnmarshalBinary(data)
	if err == nil {
		t.Fatal("expected error for bad key length")
	}
}

func TestUnmarshalBinary_Truncated(t *testing.T) {
	original, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	data := original.MarshalBinary()
	// Truncate the data
	_, err = UnmarshalBinary(data[:len(data)-10])
	if err == nil {
		t.Fatal("expected error for truncated data")
	}
}

func TestMarshalBinary_Size(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	data := id.MarshalBinary()
	expectedSize := 4 + ed25519.PrivateKeySize
	if len(data) != expectedSize {
		t.Fatalf("marshaled size = %d, want %d", len(data), expectedSize)
	}
}
