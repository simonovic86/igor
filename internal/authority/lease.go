// SPDX-License-Identifier: Apache-2.0

package authority

import (
	"fmt"
	"time"
)

// LeaseConfig holds lease timing parameters.
type LeaseConfig struct {
	Duration      time.Duration // Lease validity period (default: 60s)
	RenewalWindow float64       // Fraction of duration at which renewal triggers (default: 0.5)
	GracePeriod   time.Duration // Additional time after expiry before recovery (default: 10s)
}

// DefaultLeaseConfig returns the default lease configuration.
func DefaultLeaseConfig() LeaseConfig {
	return LeaseConfig{
		Duration:      60 * time.Second,
		RenewalWindow: 0.5,
		GracePeriod:   10 * time.Second,
	}
}

// Validate checks that lease config parameters are sensible.
func (c LeaseConfig) Validate() error {
	if c.Duration <= 0 {
		return fmt.Errorf("lease duration must be positive, got %v", c.Duration)
	}
	if c.RenewalWindow <= 0 || c.RenewalWindow >= 1.0 {
		return fmt.Errorf("renewal window must be in (0, 1), got %f", c.RenewalWindow)
	}
	if c.GracePeriod < 0 {
		return fmt.Errorf("grace period must be non-negative, got %v", c.GracePeriod)
	}
	return nil
}

// Lease tracks the time-bounded authority grant for an agent.
type Lease struct {
	Epoch  Epoch     // Current authority epoch
	Expiry time.Time // When the lease expires (zero = no lease)
	State  State     // Current authority state
	config LeaseConfig
	clock  func() time.Time // injectable clock for testing
}

// NewLease creates a new initial lease (first-time agent load).
// Bootstrap: the loading node grants itself a lease at epoch (0, 0).
func NewLease(cfg LeaseConfig) *Lease {
	now := time.Now()
	return &Lease{
		Epoch:  Epoch{MajorVersion: 0, LeaseGeneration: 0},
		Expiry: now.Add(cfg.Duration),
		State:  StateActiveOwner,
		config: cfg,
	}
}

// NewLeaseFromMigration creates a lease for a migrated agent.
// The target gets epoch (sourceMajor+1, 0) with a fresh lease.
func NewLeaseFromMigration(sourceMajor uint64, cfg LeaseConfig) *Lease {
	now := time.Now()
	return &Lease{
		Epoch:  Epoch{MajorVersion: sourceMajor + 1, LeaseGeneration: 0},
		Expiry: now.Add(cfg.Duration),
		State:  StateActiveOwner,
		config: cfg,
	}
}

// NewLeaseFromCheckpoint restores lease state from checkpoint data.
// A fresh lease duration is granted from now (the node is resuming authority).
func NewLeaseFromCheckpoint(epoch Epoch, cfg LeaseConfig) *Lease {
	now := time.Now()
	return &Lease{
		Epoch:  epoch,
		Expiry: now.Add(cfg.Duration),
		State:  StateActiveOwner,
		config: cfg,
	}
}

// now returns the current time, using the injectable clock if set.
func (l *Lease) now() time.Time {
	if l.clock != nil {
		return l.clock()
	}
	return time.Now()
}

// IsExpired returns true if the lease has expired (not counting grace period).
func (l *Lease) IsExpired() bool {
	return !l.Expiry.IsZero() && l.now().After(l.Expiry)
}

// IsExpiredWithGrace returns true if the lease + grace period has elapsed.
// This is the point at which the agent must transition to RECOVERY_REQUIRED.
func (l *Lease) IsExpiredWithGrace() bool {
	return !l.Expiry.IsZero() && l.now().After(l.Expiry.Add(l.config.GracePeriod))
}

// NeedsRenewal returns true if the remaining lease time is within the
// renewal window (i.e., less than RenewalWindow fraction remaining).
func (l *Lease) NeedsRenewal() bool {
	if l.Expiry.IsZero() {
		return false
	}
	remaining := l.Expiry.Sub(l.now())
	threshold := time.Duration(float64(l.config.Duration) * l.config.RenewalWindow)
	return remaining <= threshold
}

// Renew extends the lease for another full duration and increments
// the lease generation. This is a local operation (no network needed).
// Returns an error if the lease is expired beyond grace or not in ACTIVE_OWNER.
func (l *Lease) Renew() error {
	if l.State != StateActiveOwner {
		return fmt.Errorf("cannot renew lease in state %s", l.State)
	}
	if l.IsExpiredWithGrace() {
		return fmt.Errorf("cannot renew: lease expired beyond grace period")
	}
	l.Epoch.LeaseGeneration++
	l.Expiry = l.now().Add(l.config.Duration)
	return nil
}

// ValidateForTick checks whether the lease permits tick execution.
// Returns nil if ticking is allowed, or an error explaining why not.
// If the lease has expired beyond the grace period, transitions to
// RECOVERY_REQUIRED (EI-6: safety over liveness).
func (l *Lease) ValidateForTick() error {
	if l.State != StateActiveOwner {
		return fmt.Errorf("tick forbidden: authority state is %s", l.State)
	}
	if l.IsExpiredWithGrace() {
		l.State = StateRecoveryRequired
		return fmt.Errorf("tick forbidden: lease expired, entered RECOVERY_REQUIRED")
	}
	return nil
}

// TransitionToHandoff moves from ACTIVE_OWNER to HANDOFF_INITIATED.
func (l *Lease) TransitionToHandoff() error {
	if l.State != StateActiveOwner {
		return fmt.Errorf("cannot initiate handoff from state %s", l.State)
	}
	l.State = StateHandoffInitiated
	return nil
}

// TransitionToRetired marks the source as retired after migration.
func (l *Lease) TransitionToRetired() error {
	if l.State != StateHandoffInitiated && l.State != StateHandoffPending {
		return fmt.Errorf("cannot retire from state %s", l.State)
	}
	l.State = StateRetired
	return nil
}

// Config returns the lease configuration.
func (l *Lease) Config() LeaseConfig {
	return l.config
}
