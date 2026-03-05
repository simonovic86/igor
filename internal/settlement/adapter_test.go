package settlement

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/simonovic86/igor/pkg/receipt"
)

func TestMockAdapter_ValidateBudget_AlwaysNil(t *testing.T) {
	adapter := NewMockAdapter(slog.Default())

	// Should always return nil regardless of inputs.
	if err := adapter.ValidateBudget(context.Background(), "agent-1", 0); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := adapter.ValidateBudget(context.Background(), "agent-2", -100); err != nil {
		t.Errorf("expected nil for negative budget, got %v", err)
	}
	if err := adapter.ValidateBudget(context.Background(), "", 1_000_000); err != nil {
		t.Errorf("expected nil for empty agent ID, got %v", err)
	}
}

func TestMockAdapter_RecordSettlement_Accumulates(t *testing.T) {
	adapter := NewMockAdapter(slog.Default())
	ctx := context.Background()

	receipts := []receipt.Receipt{
		{AgentID: "a1", NodeID: "n1", EpochStart: 1, EpochEnd: 5, CostMicrocents: 100},
		{AgentID: "a1", NodeID: "n1", EpochStart: 6, EpochEnd: 10, CostMicrocents: 200},
		{AgentID: "a2", NodeID: "n2", EpochStart: 1, EpochEnd: 3, CostMicrocents: 50},
	}

	for _, r := range receipts {
		if err := adapter.RecordSettlement(ctx, r); err != nil {
			t.Fatalf("RecordSettlement failed: %v", err)
		}
	}

	got := adapter.Settlements()
	if len(got) != 3 {
		t.Fatalf("expected 3 settlements, got %d", len(got))
	}
	for i, r := range got {
		if r.AgentID != receipts[i].AgentID || r.CostMicrocents != receipts[i].CostMicrocents {
			t.Errorf("settlement %d mismatch: got %+v, want %+v", i, r, receipts[i])
		}
	}
}

func TestMockAdapter_Settlements_ReturnsCopy(t *testing.T) {
	adapter := NewMockAdapter(slog.Default())
	ctx := context.Background()

	_ = adapter.RecordSettlement(ctx, receipt.Receipt{AgentID: "a1", CostMicrocents: 100})

	s1 := adapter.Settlements()
	s1[0].CostMicrocents = 999

	s2 := adapter.Settlements()
	if s2[0].CostMicrocents != 100 {
		t.Error("Settlements() did not return a copy; mutation leaked")
	}
}

func TestMockAdapter_ConcurrentSafe(t *testing.T) {
	adapter := NewMockAdapter(slog.Default())
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = adapter.ValidateBudget(ctx, "agent", 1000)
		}()
		go func() {
			defer wg.Done()
			_ = adapter.RecordSettlement(ctx, receipt.Receipt{AgentID: "agent"})
		}()
	}
	wg.Wait()

	got := adapter.Settlements()
	if len(got) != 100 {
		t.Errorf("expected 100 settlements, got %d", len(got))
	}
}

// Compile-time interface check.
var _ BudgetAdapter = (*MockAdapter)(nil)
