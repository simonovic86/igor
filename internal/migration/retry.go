// SPDX-License-Identifier: Apache-2.0

package migration

import (
	"errors"
	"math"
	"strings"
	"time"
)

// RetryConfig controls migration retry behavior.
type RetryConfig struct {
	MaxAttempts   int           // Maximum attempts per target (default: 3)
	InitialDelay  time.Duration // First retry delay (default: 1s)
	MaxDelay      time.Duration // Cap on backoff delay (default: 30s)
	BackoffFactor float64       // Multiplier per retry (default: 2.0)
}

// DefaultRetryConfig returns sensible defaults for research use.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}
}

// Migration error sentinels for retry classification.
var (
	// ErrTransferSent indicates the transfer was sent but confirmation was not
	// received. The target MAY have the agent. Per FS-2 (Migration Continuity),
	// this is an ambiguous case — retrying to a different target risks dual
	// authority. Only retry to the same target, or enter RECOVERY_REQUIRED.
	ErrTransferSent = errors.New("transfer sent but confirmation not received")
)

// IsRetriable returns true if the migration error should be retried.
// Fatal errors (capability mismatch, hash mismatch, migration disabled,
// budget exhausted) are not retriable.
// Ambiguous errors (transfer sent, no confirmation) are not retriable.
func IsRetriable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrTransferSent) {
		return false
	}
	msg := err.Error()
	// Fatal: target explicitly rejected for permanent reasons.
	fatalPrefixes := []string{
		"WASM hash",
		"WASM binary hash mismatch",
		"capability check failed",
		"agent migration policy disabled",
		"lease handoff transition",
	}
	for _, prefix := range fatalPrefixes {
		if strings.Contains(msg, prefix) {
			return false
		}
	}
	// Fatal: target's error messages indicating permanent rejection.
	if strings.HasPrefix(msg, "target failed to start agent: ") {
		targetMsg := strings.TrimPrefix(msg, "target failed to start agent: ")
		for _, prefix := range fatalPrefixes {
			if strings.Contains(targetMsg, prefix) {
				return false
			}
		}
	}
	return true
}

// IsAmbiguous returns true if the error represents the FS-2 ambiguous case:
// transfer was sent but no confirmation received. The target MAY have the agent.
func IsAmbiguous(err error) bool {
	return err != nil && errors.Is(err, ErrTransferSent)
}

// BackoffDelay calculates the delay for attempt n (0-indexed).
func BackoffDelay(cfg RetryConfig, attempt int) time.Duration {
	delay := float64(cfg.InitialDelay) * math.Pow(cfg.BackoffFactor, float64(attempt))
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}
	return time.Duration(delay)
}
