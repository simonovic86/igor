// SPDX-License-Identifier: Apache-2.0

// Package lineage provides cryptographic types for signed checkpoint chains.
// Each checkpoint is signed by the agent's Ed25519 identity and includes the
// hash of the previous checkpoint, forming a tamper-evident lineage (EI-3).
// Content hashing uses SHA-256 for CID/IPFS compatibility.
package lineage

import (
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"fmt"
)

const (
	// PrevHashSize is the size of the previous checkpoint hash (SHA-256).
	PrevHashSize = 32

	// PublicKeySize is the Ed25519 public key size.
	PublicKeySize = ed25519.PublicKeySize // 32

	// SignatureSize is the Ed25519 signature size.
	SignatureSize = ed25519.SignatureSize // 64

	// ExtensionSize is the total bytes added by lineage fields in the v0x04 header:
	// prevHash(32) + agentPubKey(32) + signature(64) = 128.
	ExtensionSize = PrevHashSize + PublicKeySize + SignatureSize // 128
)

// ContentHash computes the SHA-256 content hash of a complete checkpoint blob.
// This hash can be used as a content address (compatible with CID raw codec + sha2-256).
func ContentHash(checkpoint []byte) [32]byte {
	return sha256.Sum256(checkpoint)
}

// SignCheckpoint signs the checkpoint signing domain (everything except the
// signature field) with the agent's private key.
func SignCheckpoint(signingDomain []byte, privKey ed25519.PrivateKey) ([SignatureSize]byte, error) {
	if len(privKey) != ed25519.PrivateKeySize {
		return [SignatureSize]byte{}, errors.New("invalid Ed25519 private key size")
	}
	sig := ed25519.Sign(privKey, signingDomain)
	var arr [SignatureSize]byte
	copy(arr[:], sig)
	return arr, nil
}

// VerifyCheckpoint verifies the signature over the checkpoint signing domain.
func VerifyCheckpoint(signingDomain []byte, pubKey ed25519.PublicKey, sig [SignatureSize]byte) bool {
	if len(pubKey) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(pubKey, signingDomain, sig[:])
}

// MarshalExtension serializes lineage fields for the v0x04 checkpoint header.
// Layout: [prevHash:32][agentPubKey:32][signature:64]
func MarshalExtension(prevHash [32]byte, pubKey ed25519.PublicKey, sig [SignatureSize]byte) []byte {
	buf := make([]byte, 0, ExtensionSize)
	buf = append(buf, prevHash[:]...)
	buf = append(buf, pubKey...)
	buf = append(buf, sig[:]...)
	return buf
}

// UnmarshalExtension parses lineage fields from checkpoint bytes starting at offset.
func UnmarshalExtension(data []byte) (prevHash [32]byte, pubKey ed25519.PublicKey, sig [SignatureSize]byte, err error) {
	if len(data) < ExtensionSize {
		return [32]byte{}, nil, [SignatureSize]byte{}, fmt.Errorf(
			"lineage extension too short: %d bytes (need %d)", len(data), ExtensionSize)
	}
	copy(prevHash[:], data[0:PrevHashSize])
	pubKey = make([]byte, PublicKeySize)
	copy(pubKey, data[PrevHashSize:PrevHashSize+PublicKeySize])
	copy(sig[:], data[PrevHashSize+PublicKeySize:ExtensionSize])
	return prevHash, pubKey, sig, nil
}

// BuildSigningDomain constructs the byte sequence that gets signed.
// It is the full checkpoint with the 64-byte signature field removed.
// Layout: [headerBeforeSig][state] — where headerBeforeSig includes everything
// up to the signature slot (version through agentPubKey), and state follows.
func BuildSigningDomain(headerBeforeSig []byte, state []byte) []byte {
	domain := make([]byte, 0, len(headerBeforeSig)+len(state))
	domain = append(domain, headerBeforeSig...)
	domain = append(domain, state...)
	return domain
}
