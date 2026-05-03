package logger

import (
	"log/slog"
	"os"
)

// Level adalah alias untuk kemudahan caller.
type Level = slog.Level

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// Init mengkonfigurasi slog global handler.
// Dipanggil sekali di main() sebelum server start.
// format: "text" (default, human-readable) atau "json" (untuk log aggregator).
func Init(level Level, format string) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{Level: level}

	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

// With mengembalikan logger dengan field tetap — untuk component-level logging.
// Contoh: log := logger.With("component", "cache")
func With(args ...any) *slog.Logger {
	return slog.With(args...)
}
