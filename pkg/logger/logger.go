// Package logger provides a structured JSON logger using stdlib slog.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New creates a new slog.Logger with JSON output.
// level accepts: "debug", "info", "warn", "error"
func New(level string) *slog.Logger {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     l,
		AddSource: l == slog.LevelDebug,
	})
	return slog.New(handler)
}
