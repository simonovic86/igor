// Package budget provides integer-based budget arithmetic for agent execution metering.
// Budget values are represented as int64 microcents (1 currency unit = 1,000,000 microcents)
// to avoid floating-point precision loss that would violate RE-3 (Budget Conservation).
package budget

import "fmt"

// MicrocentScale is the number of microcents per currency unit.
const MicrocentScale int64 = 1_000_000

// FromFloat converts a floating-point currency amount to int64 microcents.
// Used at system boundaries (CLI input, config loading).
func FromFloat(f float64) int64 {
	return int64(f * float64(MicrocentScale))
}

// Format renders microcents as a human-readable string with 6 decimal places.
func Format(microcents int64) string {
	sign := ""
	if microcents < 0 {
		sign = "-"
		microcents = -microcents
	}
	whole := microcents / MicrocentScale
	frac := microcents % MicrocentScale
	return fmt.Sprintf("%s%d.%06d", sign, whole, frac)
}
