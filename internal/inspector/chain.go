// SPDX-License-Identifier: Apache-2.0

package inspector

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/pkg/identity"
	"github.com/simonovic86/igor/pkg/lineage"
)

// ChainResult holds the result of verifying an agent's full checkpoint lineage.
type ChainResult struct {
	AgentDID       string
	AgentPubKeyHex string
	Checkpoints    int
	FirstTick      uint64
	LastTick       uint64
	ChainValid     bool
	Errors         []string
	// Segments tracks execution across different WASM binaries (migration points).
	Segments []ChainSegment
}

// ChainSegment represents a contiguous run of checkpoints with the same WASM hash.
type ChainSegment struct {
	WASMHashHex string
	StartTick   uint64
	EndTick     uint64
	Count       int
}

// chainState tracks mutable state across checkpoint verification iterations.
type chainState struct {
	prevContentHash [32]byte
	prevTick        uint64
	currentSegment  *ChainSegment
	agentPubKey     ed25519.PublicKey
}

// VerifyChain walks a checkpoint history directory and verifies the full
// cryptographic lineage: each checkpoint's signature, and prevHash continuity.
func VerifyChain(historyDir string) (*ChainResult, error) {
	files, err := listCheckpointFiles(historyDir)
	if err != nil {
		return nil, err
	}

	result := &ChainResult{ChainValid: true}
	state := &chainState{}

	for i, f := range files {
		data, readErr := os.ReadFile(f)
		if readErr != nil {
			result.addError("failed to read %s: %v", filepath.Base(f), readErr)
			continue
		}

		hdr, agentState, parseErr := agent.ParseCheckpointHeader(data)
		if parseErr != nil {
			result.addError("failed to parse %s: %v", filepath.Base(f), parseErr)
			continue
		}

		result.Checkpoints++
		verifyCheckpoint(result, state, i, data, hdr, agentState)
	}

	if len(result.Errors) > 0 {
		result.ChainValid = false
	}

	return result, nil
}

// listCheckpointFiles returns sorted .ckpt file paths from the history directory.
func listCheckpointFiles(historyDir string) ([]string, error) {
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		return nil, fmt.Errorf("read history directory: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".ckpt") {
			files = append(files, filepath.Join(historyDir, e.Name()))
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no checkpoint files found in %s", historyDir)
	}

	sort.Strings(files) // sorted by tick number (zero-padded filenames)
	return files, nil
}

// verifyCheckpoint validates a single checkpoint against the chain state.
func verifyCheckpoint(result *ChainResult, cs *chainState, i int, data []byte, hdr *agent.CheckpointHeader, agentState []byte) {
	if i == 0 {
		result.FirstTick = hdr.TickNumber
		if hdr.HasLineage {
			cs.agentPubKey = hdr.AgentPubKey
			result.AgentPubKeyHex = hex.EncodeToString(cs.agentPubKey)
			id := &identity.AgentIdentity{PublicKey: cs.agentPubKey}
			result.AgentDID = id.DID()
		}
	}
	result.LastTick = hdr.TickNumber

	if i > 0 && hdr.TickNumber <= cs.prevTick {
		result.addError("tick %d: non-monotonic (prev was %d)", hdr.TickNumber, cs.prevTick)
	}

	verifyLineage(result, cs, i, data, hdr, agentState)
	updateSegments(result, cs, hdr)

	cs.prevContentHash = lineage.ContentHash(data)
	cs.prevTick = hdr.TickNumber
}

// verifyLineage checks signature, prevHash chain, and consistent identity.
func verifyLineage(result *ChainResult, cs *chainState, i int, data []byte, hdr *agent.CheckpointHeader, agentState []byte) {
	if !hdr.HasLineage {
		if i > 0 {
			result.addError("tick %d: missing lineage (not v4)", hdr.TickNumber)
		}
		return
	}

	signingDomain := lineage.BuildSigningDomain(data[:145], agentState)
	if !lineage.VerifyCheckpoint(signingDomain, hdr.AgentPubKey, hdr.Signature) {
		result.addError("tick %d: INVALID signature", hdr.TickNumber)
	}

	if i > 0 {
		if hdr.PrevHash != cs.prevContentHash {
			result.addError("tick %d: prevHash mismatch (expected %s, got %s)",
				hdr.TickNumber,
				hex.EncodeToString(cs.prevContentHash[:8])+"...",
				hex.EncodeToString(hdr.PrevHash[:8])+"...",
			)
		}
		if !hdr.AgentPubKey.Equal(cs.agentPubKey) {
			result.addError("tick %d: agent identity changed", hdr.TickNumber)
		}
	}
}

// updateSegments tracks WASM hash segments across checkpoints.
func updateSegments(result *ChainResult, cs *chainState, hdr *agent.CheckpointHeader) {
	wasmHex := hex.EncodeToString(hdr.WASMHash[:])
	if cs.currentSegment == nil || cs.currentSegment.WASMHashHex != wasmHex {
		cs.currentSegment = &ChainSegment{
			WASMHashHex: wasmHex,
			StartTick:   hdr.TickNumber,
			EndTick:     hdr.TickNumber,
			Count:       1,
		}
		result.Segments = append(result.Segments, *cs.currentSegment)
	} else {
		cs.currentSegment.EndTick = hdr.TickNumber
		cs.currentSegment.Count++
		result.Segments[len(result.Segments)-1] = *cs.currentSegment
	}
}

func (r *ChainResult) addError(format string, args ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
}

// PrintChain writes a human-readable chain verification report.
func (r *ChainResult) PrintChain(w io.Writer) {
	fmt.Fprintf(w, "Checkpoint Lineage Verifier\n")
	fmt.Fprintf(w, "===========================\n\n")

	if r.AgentDID != "" {
		fmt.Fprintf(w, "Agent:          %s\n", r.AgentDID)
	}
	if r.AgentPubKeyHex != "" {
		fmt.Fprintf(w, "Public Key:     %s\n", r.AgentPubKeyHex)
	}
	fmt.Fprintf(w, "Checkpoints:    %d\n", r.Checkpoints)
	fmt.Fprintf(w, "Tick Range:     %d → %d\n", r.FirstTick, r.LastTick)

	if len(r.Segments) > 0 {
		fmt.Fprintf(w, "\nExecution Segments:\n")
		for i, seg := range r.Segments {
			fmt.Fprintf(w, "  [%d] WASM %s...  ticks %d→%d (%d checkpoints)\n",
				i+1, seg.WASMHashHex[:16], seg.StartTick, seg.EndTick, seg.Count)
		}
	}

	fmt.Fprintln(w)
	if r.ChainValid {
		fmt.Fprintf(w, "Lineage:        VALID (all %d signatures verified, chain unbroken)\n", r.Checkpoints)
	} else {
		fmt.Fprintf(w, "Lineage:        INVALID (%d errors)\n", len(r.Errors))
		for _, e := range r.Errors {
			fmt.Fprintf(w, "  - %s\n", e)
		}
	}
}
