package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.NodeID == "" {
		t.Error("NodeID should not be empty")
	}
	if cfg.ListenAddress != "/ip4/0.0.0.0/tcp/4001" {
		t.Errorf("ListenAddress: got %q, want /ip4/0.0.0.0/tcp/4001", cfg.ListenAddress)
	}
	if cfg.PricePerSecond != 1000 {
		t.Errorf("PricePerSecond: got %d, want 1000", cfg.PricePerSecond)
	}
	if cfg.CheckpointDir != "./checkpoints" {
		t.Errorf("CheckpointDir: got %q, want ./checkpoints", cfg.CheckpointDir)
	}
	if cfg.ReplayWindowSize != 16 {
		t.Errorf("ReplayWindowSize: got %d, want 16", cfg.ReplayWindowSize)
	}
	if cfg.VerifyInterval != 5 {
		t.Errorf("VerifyInterval: got %d, want 5", cfg.VerifyInterval)
	}
}

func TestConfig_Validate_DefaultsAreValid(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should be valid: %v", err)
	}
}

func TestConfig_Validate_ZeroPrice(t *testing.T) {
	cfg := &Config{PricePerSecond: 0, ReplayMode: "full", ReplayOnDivergence: "log"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for zero PricePerSecond")
	}
}

func TestConfig_Validate_NegativePrice(t *testing.T) {
	cfg := &Config{PricePerSecond: -1, ReplayMode: "full", ReplayOnDivergence: "log"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for negative PricePerSecond")
	}
}

func TestConfig_Validate_InvalidReplayMode(t *testing.T) {
	cfg := &Config{PricePerSecond: 1000, ReplayMode: "invalid", ReplayOnDivergence: "log"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid ReplayMode")
	}
}

func TestConfig_Validate_NegativeReplayWindow(t *testing.T) {
	cfg := &Config{PricePerSecond: 1000, ReplayWindowSize: -1, ReplayMode: "full", ReplayOnDivergence: "log"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for negative ReplayWindowSize")
	}
}

func TestConfig_Validate_NegativeVerifyInterval(t *testing.T) {
	cfg := &Config{PricePerSecond: 1000, VerifyInterval: -1, ReplayMode: "full", ReplayOnDivergence: "log"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for negative VerifyInterval")
	}
}

func TestConfig_Validate_InvalidReplayOnDivergence(t *testing.T) {
	cfg := &Config{PricePerSecond: 1000, ReplayMode: "full", ReplayOnDivergence: "invalid"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid ReplayOnDivergence")
	}
}

func TestConfig_Validate_AllDivergencePolicies(t *testing.T) {
	for _, policy := range []string{"log", "pause", "intensify", "migrate"} {
		cfg := &Config{PricePerSecond: 1000, ReplayMode: "full", ReplayOnDivergence: policy}
		if err := cfg.Validate(); err != nil {
			t.Errorf("policy %q should be valid: %v", policy, err)
		}
	}
}

func TestLoad_DefaultReplayOnDivergence(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ReplayOnDivergence != "log" {
		t.Errorf("ReplayOnDivergence: got %q, want %q", cfg.ReplayOnDivergence, "log")
	}
}
