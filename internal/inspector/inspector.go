// SPDX-License-Identifier: Apache-2.0

// Package inspector provides checkpoint file parsing and display.
package inspector

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"time"

	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/internal/authority"
	"github.com/simonovic86/igor/pkg/budget"
)

// Result holds parsed checkpoint information.
type Result struct {
	Version         byte
	Budget          int64
	BudgetFormatted string
	PricePerSecond  int64
	PriceFormatted  string
	TickNumber      uint64
	WASMHash        [32]byte
	WASMHashHex     string
	Epoch           authority.Epoch
	LeaseExpiry     int64 // Unix nanoseconds; 0 = no lease
	StateSize       int
	State           []byte
	TotalSize       int
	WASMVerified    *bool
	WASMPath        string
}

// InspectFile parses a checkpoint file and returns structured results.
func InspectFile(path string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checkpoint: %w", err)
	}
	return Inspect(data)
}

// Inspect parses raw checkpoint bytes.
func Inspect(data []byte) (*Result, error) {
	budgetVal, price, tick, wasmHash, epoch, leaseExpiry, state, err := agent.ParseCheckpointHeader(data)
	if err != nil {
		return nil, fmt.Errorf("parse checkpoint: %w", err)
	}

	return &Result{
		Version:         data[0],
		Budget:          budgetVal,
		BudgetFormatted: budget.Format(budgetVal),
		PricePerSecond:  price,
		PriceFormatted:  budget.Format(price),
		TickNumber:      tick,
		WASMHash:        wasmHash,
		WASMHashHex:     hex.EncodeToString(wasmHash[:]),
		Epoch:           epoch,
		LeaseExpiry:     leaseExpiry,
		StateSize:       len(state),
		State:           state,
		TotalSize:       len(data),
	}, nil
}

// VerifyWASM checks if a WASM binary matches the checkpoint's stored hash.
func (r *Result) VerifyWASM(wasmPath string) error {
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return fmt.Errorf("read WASM: %w", err)
	}
	hash := sha256.Sum256(wasmBytes)
	verified := hash == r.WASMHash
	r.WASMVerified = &verified
	r.WASMPath = wasmPath
	if !verified {
		return fmt.Errorf("WASM hash mismatch: checkpoint=%s binary=%s",
			r.WASMHashHex, hex.EncodeToString(hash[:]))
	}
	return nil
}

// Print writes a human-readable inspection report.
func (r *Result) Print(w io.Writer) {
	fmt.Fprintf(w, "Checkpoint Inspector\n")
	fmt.Fprintf(w, "====================\n\n")
	fmt.Fprintf(w, "Version:          %d (0x%02x)\n", r.Version, r.Version)
	fmt.Fprintf(w, "Budget:           %s (%d microcents)\n", r.BudgetFormatted, r.Budget)
	fmt.Fprintf(w, "Price/Second:     %s (%d microcents)\n", r.PriceFormatted, r.PricePerSecond)
	fmt.Fprintf(w, "Tick Number:      %d\n", r.TickNumber)
	fmt.Fprintf(w, "WASM Hash:        %s\n", r.WASMHashHex)
	if r.Version >= 0x03 {
		fmt.Fprintf(w, "Epoch:            %s\n", r.Epoch)
		if r.LeaseExpiry > 0 {
			fmt.Fprintf(w, "Lease Expiry:     %s\n", time.Unix(0, r.LeaseExpiry).UTC().Format(time.RFC3339))
		} else {
			fmt.Fprintf(w, "Lease Expiry:     (none)\n")
		}
		fmt.Fprintf(w, "Header Size:      81 bytes\n")
	} else {
		fmt.Fprintf(w, "Header Size:      57 bytes\n")
	}
	fmt.Fprintf(w, "Total Size:       %d bytes\n", r.TotalSize)
	fmt.Fprintf(w, "State Size:       %d bytes\n", r.StateSize)

	if r.WASMVerified != nil {
		if *r.WASMVerified {
			fmt.Fprintf(w, "WASM Verified:    YES (matches %s)\n", r.WASMPath)
		} else {
			fmt.Fprintf(w, "WASM Verified:    NO (mismatch with %s)\n", r.WASMPath)
		}
	}

	if r.StateSize > 0 {
		fmt.Fprintf(w, "\nState Hex Dump:\n")
		limit := r.StateSize
		if limit > 256 {
			limit = 256
		}
		printHexDump(w, r.State[:limit])
		if r.StateSize > 256 {
			fmt.Fprintf(w, "  ... (%d more bytes)\n", r.StateSize-256)
		}
	}
}

// printHexDump writes a canonical hex dump (16 bytes per line).
func printHexDump(w io.Writer, data []byte) {
	for i := 0; i < len(data); i += 16 {
		end := i + 16
		if end > len(data) {
			end = len(data)
		}
		row := data[i:end]

		// Offset
		fmt.Fprintf(w, "  %08x  ", i)

		// Hex bytes
		for j, b := range row {
			fmt.Fprintf(w, "%02x ", b)
			if j == 7 {
				fmt.Fprint(w, " ")
			}
		}
		// Pad remaining
		for j := len(row); j < 16; j++ {
			fmt.Fprint(w, "   ")
			if j == 7 {
				fmt.Fprint(w, " ")
			}
		}

		// ASCII
		fmt.Fprint(w, " |")
		for _, b := range row {
			if b >= 0x20 && b <= 0x7e {
				fmt.Fprintf(w, "%c", b)
			} else {
				fmt.Fprint(w, ".")
			}
		}
		fmt.Fprintln(w, "|")
	}
}
