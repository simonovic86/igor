// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"fmt"
	"math/big"
)

// base58btc alphabet used by did:key encoding.
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// DID returns the agent's decentralized identifier as a did:key string.
// Encoding: did:key:z + base58btc(multicodec_ed25519_pub ++ raw_public_key).
// The multicodec prefix for Ed25519 public keys is 0xed01 (varint-encoded).
// See: https://w3c-ccg.github.io/did-method-key/
func (id *AgentIdentity) DID() string {
	// multicodec prefix for ed25519-pub: 0xed, 0x01
	buf := make([]byte, 0, 2+len(id.PublicKey))
	buf = append(buf, 0xed, 0x01)
	buf = append(buf, id.PublicKey...)
	return "did:key:z" + base58Encode(buf)
}

// DIDShort returns a truncated DID for display: did:key:z6Mk...Xf3q
func (id *AgentIdentity) DIDShort() string {
	did := id.DID()
	if len(did) <= 24 {
		return did
	}
	return did[:16] + "..." + did[len(did)-4:]
}

// ParseDID extracts the raw Ed25519 public key from a did:key string.
// Returns an error if the DID is malformed or uses a different key type.
func ParseDID(did string) ([]byte, error) {
	const prefix = "did:key:z"
	if len(did) <= len(prefix) {
		return nil, fmt.Errorf("invalid did:key: too short")
	}
	if did[:len(prefix)] != prefix {
		return nil, fmt.Errorf("invalid did:key: missing 'did:key:z' prefix")
	}
	decoded, err := base58Decode(did[len(prefix):])
	if err != nil {
		return nil, fmt.Errorf("invalid did:key: base58 decode: %w", err)
	}
	if len(decoded) < 2 {
		return nil, fmt.Errorf("invalid did:key: decoded too short")
	}
	if decoded[0] != 0xed || decoded[1] != 0x01 {
		return nil, fmt.Errorf("invalid did:key: not an Ed25519 key (got multicodec 0x%02x%02x)", decoded[0], decoded[1])
	}
	pubKey := decoded[2:]
	if len(pubKey) != 32 {
		return nil, fmt.Errorf("invalid did:key: Ed25519 key must be 32 bytes, got %d", len(pubKey))
	}
	return pubKey, nil
}

// base58Encode encodes bytes to base58btc string.
func base58Encode(data []byte) string {
	x := new(big.Int).SetBytes(data)
	base := big.NewInt(58)
	zero := big.NewInt(0)
	mod := new(big.Int)

	var result []byte
	for x.Cmp(zero) > 0 {
		x.DivMod(x, base, mod)
		result = append(result, base58Alphabet[mod.Int64()])
	}

	// Leading zero bytes → leading '1's
	for _, b := range data {
		if b != 0 {
			break
		}
		result = append(result, base58Alphabet[0])
	}

	// Reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

// base58Decode decodes a base58btc string to bytes.
func base58Decode(s string) ([]byte, error) {
	result := big.NewInt(0)
	base := big.NewInt(58)

	for _, c := range s {
		idx := -1
		for i, a := range base58Alphabet {
			if a == c {
				idx = i
				break
			}
		}
		if idx < 0 {
			return nil, fmt.Errorf("invalid base58 character: %c", c)
		}
		result.Mul(result, base)
		result.Add(result, big.NewInt(int64(idx)))
	}

	decoded := result.Bytes()

	// Restore leading zeros
	numLeadingZeros := 0
	for _, c := range s {
		if c != rune(base58Alphabet[0]) {
			break
		}
		numLeadingZeros++
	}
	if numLeadingZeros > 0 {
		decoded = append(make([]byte, numLeadingZeros), decoded...)
	}

	return decoded, nil
}
