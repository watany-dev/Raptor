// Package logger provides structured logging using log/slog.
package logger

import (
	"io"
	"log/slog"
	"os"
)

// Init initializes the default slog logger with text handler writing to stderr.
func Init() {
	InitWithWriter(os.Stderr)
}

// InitWithWriter initializes the default slog logger with text handler writing to the given writer.
func InitWithWriter(w io.Writer) {
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}

// InitWithLevel initializes the default slog logger with the given level.
func InitWithLevel(w io.Writer, level slog.Level) {
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}
