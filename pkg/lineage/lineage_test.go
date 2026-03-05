// SPDX-License-Identifier: Apache-2.0

package lineage

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func generateTestKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey error: %v", err)
	}
	return pub, priv
}

func TestSignVerifyRoundTrip(t *testing.T) {
	pub, priv := generateTestKey(t)
	domain := []byte("checkpoint signing domain data here")

	sig, err := SignCheckpoint(domain, priv)
	if err != nil {
		t.Fatalf("SignCheckpoint error: %v", err)
	}
	if !VerifyCheckpoint(domain, pub, sig) {
		t.Fatal("VerifyCheckpoint failed for valid signature")
	}
}

func TestVerify_WrongKey(t *testing.T) {
	_, priv := generateTestKey(t)
	otherPub, _ := generateTestKey(t)
	domain := []byte("checkpoint data")

	sig, err := SignCheckpoint(domain, priv)
	if err != nil {
		t.Fatalf("SignCheckpoint error: %v", err)
	}
	if VerifyCheckpoint(domain, otherPub, sig) {
		t.Fatal("VerifyCheckpoint should fail with wrong key")
	}
}

func TestVerify_TamperedData(t *testing.T) {
	pub, priv := generateTestKey(t)
	domain := []byte("checkpoint data")

	sig, err := SignCheckpoint(domain, priv)
	if err != nil {
		t.Fatalf("SignCheckpoint error: %v", err)
	}

	tampered := make([]byte, len(domain))
	copy(tampered, domain)
	tampered[0] ^= 0xFF

	if VerifyCheckpoint(tampered, pub, sig) {
		t.Fatal("VerifyCheckpoint should fail with tampered data")
	}
}

func TestVerify_BadPubKeySize(t *testing.T) {
	_, priv := generateTestKey(t)
	domain := []byte("data")
	sig, _ := SignCheckpoint(domain, priv)

	if VerifyCheckpoint(domain, ed25519.PublicKey(make([]byte, 10)), sig) {
		t.Fatal("VerifyCheckpoint should fail with bad pubkey size")
	}
}

func TestSignCheckpoint_BadKeySize(t *testing.T) {
	_, err := SignCheckpoint([]byte("data"), ed25519.PrivateKey(make([]byte, 10)))
	if err == nil {
		t.Fatal("expected error for bad private key size")
	}
}

func TestContentHash_Deterministic(t *testing.T) {
	data := []byte("identical checkpoint data")
	h1 := ContentHash(data)
	h2 := ContentHash(data)
	if h1 != h2 {
		t.Fatal("ContentHash produced different results for identical input")
	}
}

func TestContentHash_Distinct(t *testing.T) {
	h1 := ContentHash([]byte("data A"))
	h2 := ContentHash([]byte("data B"))
	if h1 == h2 {
		t.Fatal("ContentHash produced same result for different inputs")
	}
}

func TestMarshalUnmarshalExtension(t *testing.T) {
	pub, priv := generateTestKey(t)
	prevHash := [32]byte{0x01, 0x02, 0x03}
	sig, err := SignCheckpoint([]byte("domain"), priv)
	if err != nil {
		t.Fatalf("SignCheckpoint error: %v", err)
	}

	data := MarshalExtension(prevHash, pub, sig)
	if len(data) != ExtensionSize {
		t.Fatalf("extension size = %d, want %d", len(data), ExtensionSize)
	}

	gotPrev, gotPub, gotSig, err := UnmarshalExtension(data)
	if err != nil {
		t.Fatalf("UnmarshalExtension error: %v", err)
	}
	if gotPrev != prevHash {
		t.Fatal("prevHash mismatch")
	}
	if !gotPub.Equal(pub) {
		t.Fatal("pubKey mismatch")
	}
	if gotSig != sig {
		t.Fatal("signature mismatch")
	}
}

func TestUnmarshalExtension_TooShort(t *testing.T) {
	_, _, _, err := UnmarshalExtension(make([]byte, ExtensionSize-1))
	if err == nil {
		t.Fatal("expected error for too-short extension data")
	}
}

func TestGenesisCheckpoint_ZeroPrevHash(t *testing.T) {
	pub, priv := generateTestKey(t)
	prevHash := [32]byte{} // all zeros for genesis
	domain := []byte("genesis checkpoint signing domain")

	sig, err := SignCheckpoint(domain, priv)
	if err != nil {
		t.Fatalf("SignCheckpoint error: %v", err)
	}

	data := MarshalExtension(prevHash, pub, sig)
	gotPrev, _, _, err := UnmarshalExtension(data)
	if err != nil {
		t.Fatalf("UnmarshalExtension error: %v", err)
	}
	if gotPrev != ([32]byte{}) {
		t.Fatal("genesis prevHash should be all zeros")
	}
}

func TestBuildSigningDomain(t *testing.T) {
	header := []byte{0x04, 0x01, 0x02, 0x03}
	state := []byte{0xAA, 0xBB}
	domain := BuildSigningDomain(header, state)
	expected := append([]byte{0x04, 0x01, 0x02, 0x03}, 0xAA, 0xBB)
	if len(domain) != len(expected) {
		t.Fatalf("domain length = %d, want %d", len(domain), len(expected))
	}
	for i := range domain {
		if domain[i] != expected[i] {
			t.Fatalf("domain[%d] = %x, want %x", i, domain[i], expected[i])
		}
	}
}
