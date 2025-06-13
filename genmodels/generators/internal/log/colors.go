package log

import (
	"context"
	"log/slog"
)

const (
	// ANSI modes
	ansiEsc          = '\u001b'
	ansiReset        = "\u001b[0m"
	ansiFaint        = "\u001b[2m"
	ansiResetFaint   = "\u001b[22m"
	ansiBrightRed    = "\u001b[91m"
	ansiBrightGreen  = "\u001b[92m"
	ansiBrightYellow = "\u001b[93m"
)

type coloredHandler struct {
	slog.TextHandler
}

func (c coloredHandler) Handle(ctx context.Context, rec slog.Record) error {
	return c.TextHandler.Handle(ctx, rec)
}

func (c coloredHandler) WithAttrs(attr []slog.Attr) slog.Handler {
	return c.TextHandler.WithAttrs(attr)
}

func (c coloredHandler) WithGroup(name string) slog.Handler {
	return c.TextHandler.WithGroup(name)
}
