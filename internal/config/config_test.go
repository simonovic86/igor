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
