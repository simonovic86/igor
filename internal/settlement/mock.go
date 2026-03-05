package settlement

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/simonovic86/igor/pkg/receipt"
)

// MockAdapter always validates budgets as valid and records settlements
// in memory. Used for testing and development without external payment rails.
type MockAdapter struct {
	mu          sync.Mutex
	settlements []receipt.Receipt
	logger      *slog.Logger
}

// NewMockAdapter creates a mock settlement adapter.
func NewMockAdapter(logger *slog.Logger) *MockAdapter {
	return &MockAdapter{logger: logger}
}

// ValidateBudget always returns nil (budget always valid in mock mode).
func (m *MockAdapter) ValidateBudget(_ context.Context, _ string, _ int64) error {
	return nil
}

// RecordSettlement stores the receipt in memory for inspection.
func (m *MockAdapter) RecordSettlement(_ context.Context, r receipt.Receipt) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settlements = append(m.settlements, r)
	m.logger.Info("Settlement recorded",
		"agent_id", r.AgentID,
		"cost", r.CostMicrocents,
		"epoch", fmt.Sprintf("%d-%d", r.EpochStart, r.EpochEnd),
	)
	return nil
}

// Settlements returns a copy of all recorded settlements.
func (m *MockAdapter) Settlements() []receipt.Receipt {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]receipt.Receipt, len(m.settlements))
	copy(out, m.settlements)
	return out
}
