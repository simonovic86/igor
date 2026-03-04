// Package receipt defines payment receipt data structures and cryptographic signing.
// A receipt is a signed attestation that a node executed an agent for a given cost
// during a checkpoint epoch. Receipts form an auditable payment trail (Phase 4).
package receipt

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
)

// Receipt is a signed attestation that a node executed an agent for a given cost.
type Receipt struct {
	AgentID        string // Agent that was executed
	NodeID         string // libp2p peer ID of the executing node (signer)
	EpochStart     uint64 // First tick number in this epoch
	EpochEnd       uint64 // Last tick number in this epoch (inclusive)
	CostMicrocents int64  // Total cost charged during this epoch
	BudgetAfter    int64  // Remaining budget after this epoch
	Timestamp      int64  // Unix nanoseconds when receipt was created
	Signature      []byte // Ed25519 signature over canonical encoding
}

// MarshalSignData produces the canonical byte sequence that gets signed.
// Layout: [agentID_len:4][agentID][nodeID_len:4][nodeID][epochStart:8]
// [epochEnd:8][costMicrocents:8][budgetAfter:8][timestamp:8]
func (r *Receipt) MarshalSignData() []byte {
	agentBytes := []byte(r.AgentID)
	nodeBytes := []byte(r.NodeID)
	size := 4 + len(agentBytes) + 4 + len(nodeBytes) + 8 + 8 + 8 + 8 + 8
	buf := make([]byte, 0, size)

	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(agentBytes)))
	buf = append(buf, agentBytes...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(nodeBytes)))
	buf = append(buf, nodeBytes...)
	buf = binary.LittleEndian.AppendUint64(buf, r.EpochStart)
	buf = binary.LittleEndian.AppendUint64(buf, r.EpochEnd)
	buf = binary.LittleEndian.AppendUint64(buf, uint64(r.CostMicrocents))
	buf = binary.LittleEndian.AppendUint64(buf, uint64(r.BudgetAfter))
	buf = binary.LittleEndian.AppendUint64(buf, uint64(r.Timestamp))

	return buf
}

// Sign signs the receipt with the given Ed25519 private key.
func (r *Receipt) Sign(key ed25519.PrivateKey) error {
	if len(key) != ed25519.PrivateKeySize {
		return errors.New("invalid Ed25519 private key size")
	}
	data := r.MarshalSignData()
	r.Signature = ed25519.Sign(key, data)
	return nil
}

// Verify checks the signature against the given Ed25519 public key.
func (r *Receipt) Verify(key ed25519.PublicKey) bool {
	if len(key) != ed25519.PublicKeySize || len(r.Signature) != ed25519.SignatureSize {
		return false
	}
	data := r.MarshalSignData()
	return ed25519.Verify(key, data, r.Signature)
}

// MarshalBinary serializes the full receipt including signature.
// Layout: [signData][sig_len:4][signature]
func (r *Receipt) MarshalBinary() []byte {
	signData := r.MarshalSignData()
	size := len(signData) + 4 + len(r.Signature)
	buf := make([]byte, 0, size)
	buf = append(buf, signData...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(r.Signature)))
	buf = append(buf, r.Signature...)
	return buf
}

// UnmarshalBinary deserializes a receipt from binary format.
func UnmarshalBinary(data []byte) (*Receipt, error) {
	if len(data) < 4 {
		return nil, errors.New("receipt too short")
	}
	pos := 0

	readUint32 := func() (uint32, error) {
		if pos+4 > len(data) {
			return 0, errors.New("receipt truncated at uint32")
		}
		v := binary.LittleEndian.Uint32(data[pos:])
		pos += 4
		return v, nil
	}

	readUint64 := func() (uint64, error) {
		if pos+8 > len(data) {
			return 0, errors.New("receipt truncated at uint64")
		}
		v := binary.LittleEndian.Uint64(data[pos:])
		pos += 8
		return v, nil
	}

	readString := func() (string, error) {
		slen, err := readUint32()
		if err != nil {
			return "", err
		}
		if pos+int(slen) > len(data) {
			return "", fmt.Errorf("receipt truncated at string (need %d, have %d)", slen, len(data)-pos)
		}
		s := string(data[pos : pos+int(slen)])
		pos += int(slen)
		return s, nil
	}

	r := &Receipt{}
	var err error

	r.AgentID, err = readString()
	if err != nil {
		return nil, fmt.Errorf("agentID: %w", err)
	}
	r.NodeID, err = readString()
	if err != nil {
		return nil, fmt.Errorf("nodeID: %w", err)
	}

	v, err := readUint64()
	if err != nil {
		return nil, fmt.Errorf("epochStart: %w", err)
	}
	r.EpochStart = v

	v, err = readUint64()
	if err != nil {
		return nil, fmt.Errorf("epochEnd: %w", err)
	}
	r.EpochEnd = v

	v, err = readUint64()
	if err != nil {
		return nil, fmt.Errorf("costMicrocents: %w", err)
	}
	r.CostMicrocents = int64(v)

	v, err = readUint64()
	if err != nil {
		return nil, fmt.Errorf("budgetAfter: %w", err)
	}
	r.BudgetAfter = int64(v)

	v, err = readUint64()
	if err != nil {
		return nil, fmt.Errorf("timestamp: %w", err)
	}
	r.Timestamp = int64(v)

	sigLen, err := readUint32()
	if err != nil {
		return nil, fmt.Errorf("sigLen: %w", err)
	}
	if pos+int(sigLen) > len(data) {
		return nil, fmt.Errorf("receipt truncated at signature (need %d, have %d)", sigLen, len(data)-pos)
	}
	r.Signature = make([]byte, sigLen)
	copy(r.Signature, data[pos:pos+int(sigLen)])

	return r, nil
}

// MarshalReceipts serializes a slice of receipts.
// Layout: [count:4][receipt1_len:4][receipt1][receipt2_len:4][receipt2]...
func MarshalReceipts(receipts []Receipt) []byte {
	buf := make([]byte, 0, 4+len(receipts)*256)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(receipts)))
	for i := range receipts {
		rb := receipts[i].MarshalBinary()
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(rb)))
		buf = append(buf, rb...)
	}
	return buf
}

// UnmarshalReceipts deserializes a slice of receipts.
func UnmarshalReceipts(data []byte) ([]Receipt, error) {
	if len(data) < 4 {
		return nil, errors.New("receipts data too short")
	}
	count := binary.LittleEndian.Uint32(data[:4])
	pos := 4
	receipts := make([]Receipt, 0, count)
	for i := uint32(0); i < count; i++ {
		if pos+4 > len(data) {
			return nil, fmt.Errorf("receipts truncated at entry %d length", i)
		}
		rlen := binary.LittleEndian.Uint32(data[pos:])
		pos += 4
		if pos+int(rlen) > len(data) {
			return nil, fmt.Errorf("receipts truncated at entry %d data", i)
		}
		r, err := UnmarshalBinary(data[pos : pos+int(rlen)])
		if err != nil {
			return nil, fmt.Errorf("receipt %d: %w", i, err)
		}
		receipts = append(receipts, *r)
		pos += int(rlen)
	}
	return receipts, nil
}
