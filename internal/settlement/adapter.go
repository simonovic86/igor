// Package settlement provides pluggable budget validation and settlement
// recording. The BudgetAdapter interface abstracts payment infrastructure
// so the runtime can gate tick execution on budget validity (EI-6) and
// record receipts for external settlement systems.
package settlement

import (
	"context"

	"github.com/simonovic86/igor/pkg/receipt"
)

// BudgetAdapter validates agent budgets and records settlement events.
// Implementations range from a no-op mock (testing) to an EVM L2
// settlement adapter (future). The runtime calls ValidateBudget before
// each tick and RecordSettlement after each checkpoint receipt.
type BudgetAdapter interface {
	// ValidateBudget checks whether the agent's budget is valid for execution.
	// Returns nil if valid, an error otherwise. Called before each tick to
	// gate execution on budget validity per EI-6 (Safety Over Liveness).
	ValidateBudget(ctx context.Context, agentID string, budget int64) error

	// RecordSettlement records a payment receipt for audit and settlement.
	// Called after each checkpoint epoch when a new receipt is created.
	// Errors are non-fatal: the runtime logs them but does not halt execution.
	RecordSettlement(ctx context.Context, r receipt.Receipt) error
}
