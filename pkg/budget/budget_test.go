// SPDX-License-Identifier: Apache-2.0

package budget

import "testing"

func TestFromFloat(t *testing.T) {
	tests := []struct {
		input float64
		want  int64
	}{
		{10.0, 10_000_000},
		{1.0, 1_000_000},
		{0.5, 500_000},
		{0.001, 1_000},
		{0.000001, 1},
		{0.0, 0},
	}
	for _, tt := range tests {
		got := FromFloat(tt.input)
		if got != tt.want {
			t.Errorf("FromFloat(%v) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{10_000_000, "10.000000"},
		{1_000_000, "1.000000"},
		{500_000, "0.500000"},
		{1_000, "0.001000"},
		{1, "0.000001"},
		{0, "0.000000"},
		{-500_000, "-0.500000"},
	}
	for _, tt := range tests {
		got := Format(tt.input)
		if got != tt.want {
			t.Errorf("Format(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
