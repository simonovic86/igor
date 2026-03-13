// SPDX-License-Identifier: Apache-2.0

package research

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/internal/runner"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestHandleDivergenceAction_None(t *testing.T) {
	ctx := context.Background()
	stop := HandleDivergenceAction(ctx, nil, nil, runner.DivergenceNone, nil, testLogger())
	if stop {
		t.Error("DivergenceNone should not stop the loop")
	}
}

func TestHandleDivergenceAction_Log(t *testing.T) {
	ctx := context.Background()
	stop := HandleDivergenceAction(ctx, nil, nil, runner.DivergenceLog, nil, testLogger())
	if stop {
		t.Error("DivergenceLog should not stop the loop")
	}
}

func TestHandleDivergenceAction_MigrateWithNilFn(t *testing.T) {
	ctx := context.Background()
	inst := &agent.Instance{AgentID: "test-agent"}
	stop := HandleDivergenceAction(ctx, inst, nil, runner.DivergenceMigrate, nil, testLogger())
	if !stop {
		t.Error("DivergenceMigrate with nil migrateFn should stop the loop")
	}
}

func TestHandleDivergenceAction_MigrateWithFn_Success(t *testing.T) {
	ctx := context.Background()
	inst := &agent.Instance{AgentID: "test-agent"}
	migrateFn := func(_ context.Context, _ string) error { return nil }
	stop := HandleDivergenceAction(ctx, inst, nil, runner.DivergenceMigrate, migrateFn, testLogger())
	if !stop {
		t.Error("successful migration should stop the loop")
	}
}

func TestHandleDivergenceAction_MigrateWithFn_Failure(t *testing.T) {
	ctx := context.Background()
	inst := &agent.Instance{AgentID: "test-agent"}
	migrateFn := func(_ context.Context, _ string) error { return fmt.Errorf("no peers") }
	stop := HandleDivergenceAction(ctx, inst, nil, runner.DivergenceMigrate, migrateFn, testLogger())
	if !stop {
		t.Error("failed migration should still stop the loop (pause fallback)")
	}
}
