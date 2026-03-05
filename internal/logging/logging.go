// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"log/slog"
	"os"
)

// NewLogger creates a new structured logger with timestamp and level.
func NewLogger() *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(handler)
}

// Info logs an info-level message.
func Info(logger *slog.Logger, msg string, args ...any) {
	logger.Info(msg, args...)
}

// Error logs an error-level message.
func Error(logger *slog.Logger, msg string, args ...any) {
	logger.Error(msg, args...)
}
