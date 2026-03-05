// SPDX-License-Identifier: Apache-2.0

// Package identity manages Ed25519 keypairs that serve as an agent's
// cryptographic identity. Agent identity is distinct from node identity
// (used by payment receipts); it travels with the agent across migrations
// and signs every checkpoint to create a verifiable lineage chain.
package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
)

// AgentIdentity holds the Ed25519 keypair for an agent.
// The public key serves as the agent's canonical cryptographic identity (OA-1).
// The private key signs checkpoints to create a verifiable lineage (EI-3).
type AgentIdentity struct {
	PublicKey  ed25519.PublicKey  // 32 bytes
	PrivateKey ed25519.PrivateKey // 64 bytes
}

// Generate creates a new random agent identity.
func Generate() (*AgentIdentity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate agent identity: %w", err)
	}
	return &AgentIdentity{PublicKey: pub, PrivateKey: priv}, nil
}

// FromPrivateKey reconstructs an AgentIdentity from a private key.
func FromPrivateKey(priv ed25519.PrivateKey) (*AgentIdentity, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid Ed25519 private key size")
	}
	pub, ok := priv.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("failed to derive public key")
	}
	return &AgentIdentity{PublicKey: pub, PrivateKey: priv}, nil
}

// MarshalBinary serializes the identity for persistent storage.
// Layout: [privkey_len:4 LE][private_key:64]
// Only the private key is stored; the public key is derived on load.
func (id *AgentIdentity) MarshalBinary() []byte {
	buf := make([]byte, 0, 4+ed25519.PrivateKeySize)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(id.PrivateKey)))
	buf = append(buf, id.PrivateKey...)
	return buf
}

// UnmarshalBinary deserializes an identity from binary format.
func UnmarshalBinary(data []byte) (*AgentIdentity, error) {
	if len(data) < 4 {
		return nil, errors.New("identity data too short")
	}
	keyLen := binary.LittleEndian.Uint32(data[:4])
	if keyLen != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("unexpected private key length: %d", keyLen)
	}
	if len(data) < 4+int(keyLen) {
		return nil, errors.New("identity data truncated")
	}
	priv := make([]byte, keyLen)
	copy(priv, data[4:4+keyLen])
	return FromPrivateKey(ed25519.PrivateKey(priv))
}
