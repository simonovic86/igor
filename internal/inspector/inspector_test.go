package inspector_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simonovic86/igor/internal/inspector"
)

func TestInspect_GoldenCheckpoint(t *testing.T) {
	result, err := inspector.InspectFile(filepath.Join("..", "agent", "testdata", "checkpoint.bin"))
	if err != nil {
		t.Fatal(err)
	}

	if result.Version != 2 {
		t.Errorf("version: got %d, want 2", result.Version)
	}
	if result.Budget != 1000000 {
		t.Errorf("budget: got %d, want 1000000", result.Budget)
	}
	if result.PricePerSecond != 1000 {
		t.Errorf("price: got %d, want 1000", result.PricePerSecond)
	}
	if result.TickNumber != 5 {
		t.Errorf("tick: got %d, want 5", result.TickNumber)
	}
	if result.StateSize != 8 {
		t.Errorf("state size: got %d, want 8", result.StateSize)
	}
	if result.TotalSize != 65 {
		t.Errorf("total size: got %d, want 65", result.TotalSize)
	}

	// Verify WASM hash matches sha256("known-wasm-binary-for-golden-test")
	expectedHash := sha256.Sum256([]byte("known-wasm-binary-for-golden-test"))
	if result.WASMHash != expectedHash {
		t.Errorf("WASM hash mismatch")
	}

	// Verify state contains counter=3
	counter := binary.LittleEndian.Uint64(result.State)
	if counter != 3 {
		t.Errorf("state counter: got %d, want 3", counter)
	}
}

func TestInspect_EmptyState(t *testing.T) {
	result, err := inspector.InspectFile(filepath.Join("..", "agent", "testdata", "checkpoint_empty_state.bin"))
	if err != nil {
		t.Fatal(err)
	}

	if result.Version != 2 {
		t.Errorf("version: got %d, want 2", result.Version)
	}
	if result.Budget != 500000 {
		t.Errorf("budget: got %d, want 500000", result.Budget)
	}
	if result.PricePerSecond != 2000 {
		t.Errorf("price: got %d, want 2000", result.PricePerSecond)
	}
	if result.TickNumber != 0 {
		t.Errorf("tick: got %d, want 0", result.TickNumber)
	}
	if result.StateSize != 0 {
		t.Errorf("state size: got %d, want 0", result.StateSize)
	}
	if result.TotalSize != 57 {
		t.Errorf("total size: got %d, want 57", result.TotalSize)
	}
}

func TestInspect_TooShort(t *testing.T) {
	_, err := inspector.Inspect([]byte{0x02, 0x01})
	if err == nil {
		t.Fatal("expected error for short checkpoint")
	}
	if !strings.Contains(err.Error(), "too short") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInspect_BadVersion(t *testing.T) {
	data := make([]byte, 57)
	data[0] = 0xFF
	_, err := inspector.Inspect(data)
	if err == nil {
		t.Fatal("expected error for bad version")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVerifyWASM_Match(t *testing.T) {
	// Create a temp WASM file with known content.
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "test.wasm")
	wasmBytes := []byte("known-wasm-binary-for-golden-test")
	if err := os.WriteFile(wasmPath, wasmBytes, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := inspector.InspectFile(filepath.Join("..", "agent", "testdata", "checkpoint.bin"))
	if err != nil {
		t.Fatal(err)
	}

	if err := result.VerifyWASM(wasmPath); err != nil {
		t.Errorf("expected match: %v", err)
	}
	if result.WASMVerified == nil || !*result.WASMVerified {
		t.Error("WASMVerified should be true")
	}
}

func TestVerifyWASM_Mismatch(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "wrong.wasm")
	if err := os.WriteFile(wasmPath, []byte("different-binary"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := inspector.InspectFile(filepath.Join("..", "agent", "testdata", "checkpoint.bin"))
	if err != nil {
		t.Fatal(err)
	}

	if err := result.VerifyWASM(wasmPath); err == nil {
		t.Error("expected mismatch error")
	}
	if result.WASMVerified == nil || *result.WASMVerified {
		t.Error("WASMVerified should be false")
	}
}

func TestPrint_Output(t *testing.T) {
	result, err := inspector.InspectFile(filepath.Join("..", "agent", "testdata", "checkpoint.bin"))
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	result.Print(&buf)
	output := buf.String()

	// Verify key fields are present in output.
	checks := []string{
		"Version:",
		"Budget:",
		"Price/Second:",
		"Tick Number:",
		"WASM Hash:",
		"State Size:",
		"Hex Dump:",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output missing %q", check)
		}
	}
}

func TestPrint_EmptyState(t *testing.T) {
	result, err := inspector.InspectFile(filepath.Join("..", "agent", "testdata", "checkpoint_empty_state.bin"))
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	result.Print(&buf)
	output := buf.String()

	// Empty state should not have hex dump.
	if strings.Contains(output, "Hex Dump") {
		t.Error("empty state should not show hex dump")
	}
}
