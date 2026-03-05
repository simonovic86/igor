// SPDX-License-Identifier: Apache-2.0

package authority

import (
	"testing"
	"time"
)

func testConfig() LeaseConfig {
	return LeaseConfig{
		Duration:      10 * time.Second,
		RenewalWindow: 0.5,
		GracePeriod:   2 * time.Second,
	}
}

func TestLeaseConfig_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		if err := testConfig().Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("zero duration", func(t *testing.T) {
		cfg := testConfig()
		cfg.Duration = 0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for zero duration")
		}
	})
	t.Run("negative duration", func(t *testing.T) {
		cfg := testConfig()
		cfg.Duration = -1 * time.Second
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for negative duration")
		}
	})
	t.Run("renewal window zero", func(t *testing.T) {
		cfg := testConfig()
		cfg.RenewalWindow = 0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for zero renewal window")
		}
	})
	t.Run("renewal window one", func(t *testing.T) {
		cfg := testConfig()
		cfg.RenewalWindow = 1.0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for renewal window >= 1")
		}
	})
	t.Run("negative grace period", func(t *testing.T) {
		cfg := testConfig()
		cfg.GracePeriod = -1 * time.Second
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for negative grace period")
		}
	})
	t.Run("zero grace period valid", func(t *testing.T) {
		cfg := testConfig()
		cfg.GracePeriod = 0
		if err := cfg.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNewLease_InitialState(t *testing.T) {
	cfg := testConfig()
	l := NewLease(cfg)

	if l.State != StateActiveOwner {
		t.Errorf("state = %s, want ACTIVE_OWNER", l.State)
	}
	if l.Epoch.MajorVersion != 0 || l.Epoch.LeaseGeneration != 0 {
		t.Errorf("epoch = %s, want (0,0)", l.Epoch)
	}
	if l.Expiry.IsZero() {
		t.Error("expiry should not be zero")
	}
	if l.IsExpired() {
		t.Error("new lease should not be expired")
	}
}

func TestNewLeaseFromMigration(t *testing.T) {
	cfg := testConfig()
	l := NewLeaseFromMigration(3, cfg)

	if l.Epoch.MajorVersion != 4 {
		t.Errorf("major version = %d, want 4", l.Epoch.MajorVersion)
	}
	if l.Epoch.LeaseGeneration != 0 {
		t.Errorf("lease generation = %d, want 0", l.Epoch.LeaseGeneration)
	}
	if l.State != StateActiveOwner {
		t.Errorf("state = %s, want ACTIVE_OWNER", l.State)
	}
}

func TestNewLeaseFromCheckpoint(t *testing.T) {
	cfg := testConfig()
	epoch := Epoch{MajorVersion: 2, LeaseGeneration: 5}
	l := NewLeaseFromCheckpoint(epoch, cfg)

	if !l.Epoch.Equal(epoch) {
		t.Errorf("epoch = %s, want %s", l.Epoch, epoch)
	}
	if l.State != StateActiveOwner {
		t.Errorf("state = %s, want ACTIVE_OWNER", l.State)
	}
	if l.IsExpired() {
		t.Error("restored lease should not be expired")
	}
}

func TestLease_NeedsRenewal(t *testing.T) {
	cfg := testConfig() // 10s duration, 0.5 renewal window
	l := NewLease(cfg)

	// Fresh lease: 10s remaining, threshold at 5s → no renewal needed
	if l.NeedsRenewal() {
		t.Error("fresh lease should not need renewal")
	}

	// Simulate time passing: 6s elapsed, 4s remaining < 5s threshold
	l.clock = func() time.Time { return l.Expiry.Add(-4 * time.Second) }
	if !l.NeedsRenewal() {
		t.Error("lease with 4s remaining (threshold 5s) should need renewal")
	}

	// Exactly at threshold
	l.clock = func() time.Time { return l.Expiry.Add(-5 * time.Second) }
	if !l.NeedsRenewal() {
		t.Error("lease at exactly threshold should need renewal")
	}

	// Just before threshold
	l.clock = func() time.Time { return l.Expiry.Add(-6 * time.Second) }
	if l.NeedsRenewal() {
		t.Error("lease with 6s remaining (threshold 5s) should not need renewal")
	}
}

func TestLease_Renew(t *testing.T) {
	cfg := testConfig()
	l := NewLease(cfg)

	if l.Epoch.LeaseGeneration != 0 {
		t.Fatalf("initial generation = %d, want 0", l.Epoch.LeaseGeneration)
	}

	if err := l.Renew(); err != nil {
		t.Fatalf("renew failed: %v", err)
	}

	if l.Epoch.LeaseGeneration != 1 {
		t.Errorf("generation after renew = %d, want 1", l.Epoch.LeaseGeneration)
	}
	if l.Epoch.MajorVersion != 0 {
		t.Errorf("major version changed to %d after renew", l.Epoch.MajorVersion)
	}
	if l.IsExpired() {
		t.Error("renewed lease should not be expired")
	}

	// Renew again
	if err := l.Renew(); err != nil {
		t.Fatalf("second renew failed: %v", err)
	}
	if l.Epoch.LeaseGeneration != 2 {
		t.Errorf("generation after second renew = %d, want 2", l.Epoch.LeaseGeneration)
	}
}

func TestLease_Renew_Forbidden_NotActiveOwner(t *testing.T) {
	cfg := testConfig()
	l := NewLease(cfg)

	l.State = StateHandoffInitiated
	if err := l.Renew(); err == nil {
		t.Fatal("expected error renewing in HANDOFF_INITIATED state")
	}

	l.State = StateRetired
	if err := l.Renew(); err == nil {
		t.Fatal("expected error renewing in RETIRED state")
	}

	l.State = StateRecoveryRequired
	if err := l.Renew(); err == nil {
		t.Fatal("expected error renewing in RECOVERY_REQUIRED state")
	}
}

func TestLease_Renew_Forbidden_ExpiredBeyondGrace(t *testing.T) {
	cfg := testConfig() // 10s duration, 2s grace
	l := NewLease(cfg)

	// Set clock past expiry + grace
	l.clock = func() time.Time { return l.Expiry.Add(3 * time.Second) }
	if err := l.Renew(); err == nil {
		t.Fatal("expected error renewing expired lease beyond grace")
	}
}

func TestLease_ValidateForTick_ActiveOwner(t *testing.T) {
	cfg := testConfig()
	l := NewLease(cfg)

	if err := l.ValidateForTick(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLease_ValidateForTick_NotActiveOwner(t *testing.T) {
	states := []State{StateHandoffInitiated, StateHandoffPending, StateRetired, StateRecoveryRequired}
	for _, s := range states {
		t.Run(s.String(), func(t *testing.T) {
			cfg := testConfig()
			l := NewLease(cfg)
			l.State = s
			if err := l.ValidateForTick(); err == nil {
				t.Errorf("expected error for state %s", s)
			}
		})
	}
}

func TestLease_ValidateForTick_Expired_TransitionsToRecovery(t *testing.T) {
	cfg := testConfig() // 10s duration, 2s grace
	l := NewLease(cfg)

	// Set clock past expiry + grace
	l.clock = func() time.Time { return l.Expiry.Add(3 * time.Second) }
	err := l.ValidateForTick()
	if err == nil {
		t.Fatal("expected error for expired lease")
	}
	if l.State != StateRecoveryRequired {
		t.Errorf("state = %s, want RECOVERY_REQUIRED", l.State)
	}
}

func TestLease_ValidateForTick_WithinGracePeriod(t *testing.T) {
	cfg := testConfig() // 10s duration, 2s grace
	l := NewLease(cfg)

	// Expired but within grace period (1s past expiry, grace is 2s)
	l.clock = func() time.Time { return l.Expiry.Add(1 * time.Second) }
	if err := l.ValidateForTick(); err != nil {
		t.Errorf("expected no error within grace period: %v", err)
	}
	if l.State != StateActiveOwner {
		t.Errorf("state = %s, want ACTIVE_OWNER (still within grace)", l.State)
	}
}

func TestLease_IsExpired(t *testing.T) {
	cfg := testConfig()
	l := NewLease(cfg)

	if l.IsExpired() {
		t.Error("fresh lease should not be expired")
	}

	// 1 second before expiry
	l.clock = func() time.Time { return l.Expiry.Add(-1 * time.Second) }
	if l.IsExpired() {
		t.Error("lease 1s before expiry should not be expired")
	}

	// 1 second after expiry
	l.clock = func() time.Time { return l.Expiry.Add(1 * time.Second) }
	if !l.IsExpired() {
		t.Error("lease 1s after expiry should be expired")
	}
}

func TestLease_IsExpiredWithGrace(t *testing.T) {
	cfg := testConfig() // grace = 2s
	l := NewLease(cfg)

	// 1s past expiry → within grace → not expired with grace
	l.clock = func() time.Time { return l.Expiry.Add(1 * time.Second) }
	if l.IsExpiredWithGrace() {
		t.Error("lease within grace period should not be expired with grace")
	}

	// 3s past expiry → beyond 2s grace → expired with grace
	l.clock = func() time.Time { return l.Expiry.Add(3 * time.Second) }
	if !l.IsExpiredWithGrace() {
		t.Error("lease beyond grace period should be expired with grace")
	}
}

func TestLease_TransitionToHandoff(t *testing.T) {
	cfg := testConfig()
	l := NewLease(cfg)

	if err := l.TransitionToHandoff(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.State != StateHandoffInitiated {
		t.Errorf("state = %s, want HANDOFF_INITIATED", l.State)
	}
}

func TestLease_TransitionToHandoff_InvalidState(t *testing.T) {
	cfg := testConfig()
	l := NewLease(cfg)
	l.State = StateRetired

	if err := l.TransitionToHandoff(); err == nil {
		t.Fatal("expected error transitioning from RETIRED to HANDOFF_INITIATED")
	}
}

func TestLease_TransitionToRetired(t *testing.T) {
	cfg := testConfig()
	l := NewLease(cfg)

	if err := l.TransitionToHandoff(); err != nil {
		t.Fatalf("handoff transition failed: %v", err)
	}
	if err := l.TransitionToRetired(); err != nil {
		t.Fatalf("retire transition failed: %v", err)
	}
	if l.State != StateRetired {
		t.Errorf("state = %s, want RETIRED", l.State)
	}
}

func TestLease_TransitionToRetired_InvalidState(t *testing.T) {
	cfg := testConfig()
	l := NewLease(cfg)

	// Cannot retire from ACTIVE_OWNER directly
	if err := l.TransitionToRetired(); err == nil {
		t.Fatal("expected error retiring from ACTIVE_OWNER")
	}
}

func TestLease_AntiClone_ExpiredCannotTick(t *testing.T) {
	cfg := testConfig()
	l := NewLease(cfg)

	// Simulate expired lease (past grace)
	l.clock = func() time.Time { return l.Expiry.Add(cfg.GracePeriod + 1*time.Second) }

	err := l.ValidateForTick()
	if err == nil {
		t.Fatal("expired lease should not allow ticking")
	}
	if l.State != StateRecoveryRequired {
		t.Errorf("state = %s, want RECOVERY_REQUIRED", l.State)
	}

	// Once in RECOVERY_REQUIRED, cannot tick even with fresh clock
	l.clock = nil // reset to real time
	err = l.ValidateForTick()
	if err == nil {
		t.Fatal("RECOVERY_REQUIRED state should prevent ticking")
	}
}

func TestLease_Config(t *testing.T) {
	cfg := testConfig()
	l := NewLease(cfg)

	if l.Config() != cfg {
		t.Error("Config() should return the lease config")
	}
}

func TestLease_RevertHandoff(t *testing.T) {
	cfg := testConfig()
	l := NewLease(cfg)

	if err := l.TransitionToHandoff(); err != nil {
		t.Fatalf("handoff transition failed: %v", err)
	}
	if l.State != StateHandoffInitiated {
		t.Fatalf("state = %s, want HANDOFF_INITIATED", l.State)
	}

	if err := l.RevertHandoff(); err != nil {
		t.Fatalf("revert handoff failed: %v", err)
	}
	if l.State != StateActiveOwner {
		t.Errorf("state = %s, want ACTIVE_OWNER", l.State)
	}

	// Can tick again after revert
	if err := l.ValidateForTick(); err != nil {
		t.Errorf("expected ticking allowed after revert: %v", err)
	}
}

func TestLease_RevertHandoff_InvalidState(t *testing.T) {
	tests := []struct {
		name  string
		state State
	}{
		{"ACTIVE_OWNER", StateActiveOwner},
		{"RETIRED", StateRetired},
		{"RECOVERY_REQUIRED", StateRecoveryRequired},
		{"HANDOFF_PENDING", StateHandoffPending},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			l := NewLease(cfg)
			l.State = tt.state

			if err := l.RevertHandoff(); err == nil {
				t.Errorf("expected error reverting from %s", tt.state)
			}
		})
	}
}

func TestLease_Recover(t *testing.T) {
	cfg := testConfig()
	l := NewLease(cfg)
	l.Epoch = Epoch{MajorVersion: 5, LeaseGeneration: 3}

	// Force into RECOVERY_REQUIRED
	l.clock = func() time.Time { return l.Expiry.Add(cfg.GracePeriod + 1*time.Second) }
	_ = l.ValidateForTick()
	if l.State != StateRecoveryRequired {
		t.Fatalf("state = %s, want RECOVERY_REQUIRED", l.State)
	}

	// Reset clock so the new lease isn't immediately expired
	l.clock = nil

	if err := l.Recover(); err != nil {
		t.Fatalf("recover failed: %v", err)
	}

	if l.State != StateActiveOwner {
		t.Errorf("state = %s, want ACTIVE_OWNER", l.State)
	}
	if l.Epoch.MajorVersion != 6 {
		t.Errorf("major version = %d, want 6 (incremented)", l.Epoch.MajorVersion)
	}
	if l.Epoch.LeaseGeneration != 0 {
		t.Errorf("lease generation = %d, want 0 (reset)", l.Epoch.LeaseGeneration)
	}
	if l.IsExpired() {
		t.Error("recovered lease should not be expired")
	}

	// Can tick after recovery
	if err := l.ValidateForTick(); err != nil {
		t.Errorf("expected ticking allowed after recovery: %v", err)
	}
}

func TestLease_Recover_InvalidState(t *testing.T) {
	tests := []struct {
		name  string
		state State
	}{
		{"ACTIVE_OWNER", StateActiveOwner},
		{"HANDOFF_INITIATED", StateHandoffInitiated},
		{"RETIRED", StateRetired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			l := NewLease(cfg)
			l.State = tt.state

			if err := l.Recover(); err == nil {
				t.Errorf("expected error recovering from %s", tt.state)
			}
		})
	}
}

func TestLease_Recover_SupersedesOldEpoch(t *testing.T) {
	cfg := testConfig()
	l := NewLease(cfg)
	oldEpoch := l.Epoch

	// Force recovery
	l.State = StateRecoveryRequired
	if err := l.Recover(); err != nil {
		t.Fatalf("recover failed: %v", err)
	}

	if !l.Epoch.Supersedes(oldEpoch) {
		t.Error("recovered epoch should supersede old epoch")
	}
}

func TestDefaultLeaseConfig(t *testing.T) {
	cfg := DefaultLeaseConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
	if cfg.Duration != 60*time.Second {
		t.Errorf("duration = %v, want 60s", cfg.Duration)
	}
	if cfg.RenewalWindow != 0.5 {
		t.Errorf("renewal window = %f, want 0.5", cfg.RenewalWindow)
	}
	if cfg.GracePeriod != 10*time.Second {
		t.Errorf("grace period = %v, want 10s", cfg.GracePeriod)
	}
}
