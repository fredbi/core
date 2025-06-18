package log

import (
	"context"
	"log/slog"
)

type Logger interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	DebugEnabled() bool
}

type ColoredLogger struct {
	options
	*slog.Logger
}

func NewColoredLogger(opts ...Option) *ColoredLogger {
	l := &ColoredLogger{
		options: optionsWithDefaults(opts),
	}
	l.Logger = slog.New(l.handler)

	return l
}

func (d *ColoredLogger) Debug(msg string, args ...any) {
	d.Debug(msg, args...)
}
func (d *ColoredLogger) Info(msg string, args ...any) {
	d.Info(msg, args...)
}
func (d *ColoredLogger) Warn(msg string, args ...any) {
	d.Warn(msg, args...)
}
func (d *ColoredLogger) Error(msg string, args ...any) {
	d.Error(msg, args...)
}

func (d *ColoredLogger) DebugEnabled() bool {
	return d.Enabled(context.Background(), slog.LevelDebug)
}
