package logger

import (
	"context"
	"log/slog"

	"github.com/isaacjuwon/httpex/pkg/core"
)

// SlogAdapter wraps a *slog.Logger to satisfy the core.Logger interface.
type SlogAdapter struct {
	l *slog.Logger
}

// NewSlogAdapter creates a new Logger adapter from a slog Logger.
func NewSlogAdapter(l *slog.Logger) core.Logger {
	return &SlogAdapter{l: l}
}

// NewDefaultLogger returns a SlogAdapter wrapping slog.Default().
func NewDefaultLogger() core.Logger {
	return NewSlogAdapter(slog.Default())
}

// Info logs a message at LevelInfo.
func (a *SlogAdapter) Info(msg string, attrs ...any) {
	a.l.Info(msg, attrs...)
}

// Error logs a message at LevelError.
func (a *SlogAdapter) Error(msg string, attrs ...any) {
	a.l.Error(msg, attrs...)
}

// Log logs a message at the given slog level.
func (a *SlogAdapter) Log(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	a.l.Log(ctx, level, msg, attrs...)
}
