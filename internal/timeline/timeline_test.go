// SPDX-License-Identifier: Apache-2.0

package timeline

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRender_ContainsHeader(t *testing.T) {
	tl := New("tx-1842", "bridge-reconciler-7f3a")
	tl.Add(Event{
		Timestamp: time.Date(2025, 1, 1, 3, 14, 5, 0, time.UTC),
		Kind:      KindStateChange,
		Summary:   "Case tx-1842 detected (source: chain-A, dest: chain-B)",
		Details:   []string{"State: DetectedPendingTransfer"},
	})

	var buf bytes.Buffer
	tl.Render(&buf)

	output := buf.String()

	if !strings.Contains(output, "INCIDENT TIMELINE") {
		t.Error("missing INCIDENT TIMELINE header")
	}
	if !strings.Contains(output, "tx-1842") {
		t.Error("missing case ID")
	}
	if !strings.Contains(output, "bridge-reconciler-7f3a") {
		t.Error("missing agent ID")
	}
	if !strings.Contains(output, "03:14:05") {
		t.Error("missing timestamp")
	}
	if !strings.Contains(output, "DetectedPendingTransfer") {
		t.Error("missing state detail")
	}
}

func TestRender_CrashMarker(t *testing.T) {
	tl := New("tx-1842", "bridge-reconciler")
	tl.Add(Event{
		Timestamp: time.Date(2025, 1, 1, 3, 14, 8, 0, time.UTC),
		Kind:      KindCrash,
		Summary:   "HOST NODE FAILURE",
		Details:   []string{"Node node-a became unreachable"},
	})

	var buf bytes.Buffer
	tl.Render(&buf)

	output := buf.String()
	if !strings.Contains(output, "\u2717") {
		t.Error("crash event should have cross marker")
	}
	if !strings.Contains(output, "HOST NODE FAILURE") {
		t.Error("missing crash summary")
	}
}

func TestRenderSafetySummary(t *testing.T) {
	tl := New("tx-1842", "bridge-reconciler")

	checks := []SafetyCheck{
		{Label: "No duplicate finalize execution", Passed: true},
		{Label: "Checkpoint integrity verified", Passed: true},
		{Label: "Replay determinism confirmed", Passed: true},
		{Label: "Single-instance invariant maintained", Passed: true},
	}

	var buf bytes.Buffer
	tl.RenderSafetySummary(&buf, checks)

	output := buf.String()
	if !strings.Contains(output, "SAFETY SUMMARY") {
		t.Error("missing SAFETY SUMMARY header")
	}
	if strings.Count(output, "\u2713") != 4 {
		t.Errorf("expected 4 checkmarks, got %d", strings.Count(output, "\u2713"))
	}
}

func TestRenderSafetySummary_FailedCheck(t *testing.T) {
	tl := New("tx-1842", "bridge-reconciler")

	checks := []SafetyCheck{
		{Label: "No duplicate finalize execution", Passed: true},
		{Label: "Replay determinism confirmed", Passed: false},
	}

	var buf bytes.Buffer
	tl.RenderSafetySummary(&buf, checks)

	output := buf.String()
	if strings.Count(output, "\u2713") != 1 {
		t.Errorf("expected 1 checkmark, got %d", strings.Count(output, "\u2713"))
	}
	if strings.Count(output, "\u2717") != 1 {
		t.Errorf("expected 1 cross, got %d", strings.Count(output, "\u2717"))
	}
}

func TestRenderComparison(t *testing.T) {
	tl := New("tx-1842", "bridge-reconciler")

	var buf bytes.Buffer
	tl.RenderComparison(&buf)

	output := buf.String()
	if !strings.Contains(output, "Naive Worker") {
		t.Error("missing naive worker section")
	}
	if !strings.Contains(output, "Igor Worker") {
		t.Error("missing Igor worker section")
	}
	if !strings.Contains(output, "double execution") {
		t.Error("missing double execution risk")
	}
}
