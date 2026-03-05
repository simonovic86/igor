// SPDX-License-Identifier: Apache-2.0

package migration

import (
	"fmt"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	if cfg.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts 3, got %d", cfg.MaxAttempts)
	}
	if cfg.InitialDelay != time.Second {
		t.Errorf("expected InitialDelay 1s, got %v", cfg.InitialDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("expected MaxDelay 30s, got %v", cfg.MaxDelay)
	}
	if cfg.BackoffFactor != 2.0 {
		t.Errorf("expected BackoffFactor 2.0, got %f", cfg.BackoffFactor)
	}
}

func TestIsRetriable_ConnectionErrors(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retriable bool
	}{
		{"connection refused", fmt.Errorf("failed to connect to target peer: connection refused"), true},
		{"stream open failed", fmt.Errorf("failed to open migration stream: protocol not supported"), true},
		{"timeout", fmt.Errorf("failed to read confirmation: context deadline exceeded"), true},
		{"send failed", fmt.Errorf("failed to send transfer: broken pipe"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetriable(tt.err); got != tt.retriable {
				t.Errorf("IsRetriable(%q) = %v, want %v", tt.err, got, tt.retriable)
			}
		})
	}
}

func TestIsRetriable_FatalErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"WASM hash mismatch", fmt.Errorf("WASM binary hash mismatch")},
		{"capability check", fmt.Errorf("target failed to start agent: capability check failed: missing rand")},
		{"migration disabled", fmt.Errorf("target failed to start agent: agent migration policy disabled")},
		{"lease transition", fmt.Errorf("lease handoff transition: cannot initiate handoff from state RETIRED")},
		{"WASM hash short", fmt.Errorf("WASM hash missing or invalid length")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsRetriable(tt.err) {
				t.Errorf("expected %q to be non-retriable", tt.err)
			}
		})
	}
}

func TestIsRetriable_Nil(t *testing.T) {
	if IsRetriable(nil) {
		t.Error("nil error should not be retriable")
	}
}

func TestIsAmbiguous(t *testing.T) {
	ambig := fmt.Errorf("failed to read confirmation: %w", ErrTransferSent)
	if !IsAmbiguous(ambig) {
		t.Error("expected ambiguous error to be detected")
	}
	if IsAmbiguous(fmt.Errorf("connection refused")) {
		t.Error("connection refused should not be ambiguous")
	}
	if IsAmbiguous(nil) {
		t.Error("nil should not be ambiguous")
	}
	if IsRetriable(ambig) {
		t.Error("ambiguous errors should not be retriable")
	}
}

func TestBackoffDelay_ExponentialIncrease(t *testing.T) {
	cfg := RetryConfig{
		InitialDelay:  time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}

	delays := []time.Duration{
		BackoffDelay(cfg, 0),
		BackoffDelay(cfg, 1),
		BackoffDelay(cfg, 2),
		BackoffDelay(cfg, 3),
	}

	expected := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
	}

	for i, d := range delays {
		if d != expected[i] {
			t.Errorf("attempt %d: expected %v, got %v", i, expected[i], d)
		}
	}
}

func TestBackoffDelay_CappedAtMax(t *testing.T) {
	cfg := RetryConfig{
		InitialDelay:  time.Second,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
	}

	// Attempt 10: 1s * 2^10 = 1024s > 5s → capped
	d := BackoffDelay(cfg, 10)
	if d != 5*time.Second {
		t.Errorf("expected capped at 5s, got %v", d)
	}
}
