package log

import (
	"io"
	"log/slog"
	"os"
)

type Option func(*options)

type options struct {
	name           string
	disabled       bool
	level          slog.Level
	w              io.Writer
	handler        slog.Handler
	handlerOptions *slog.HandlerOptions
}

func optionsWithDefaults(opts []Option) options {
	o := options{
		level: slog.LevelInfo,
		w:     os.Stdout,
	}

	for _, apply := range opts {
		apply(&o)
	}

	if o.handler == nil {
		if o.disabled {
			o.handler = slog.DiscardHandler

			return o
		}

		if o.handlerOptions == nil {
			o.handlerOptions = &slog.HandlerOptions{
				Level: o.level,
			}
			o.handlerOptions.AddSource = (o.level.Level() == slog.LevelDebug)
		} else {
			if o.handlerOptions.Level == nil {
				o.handlerOptions.Level = o.level
			}
		}
		o.handler = slog.NewTextHandler(o.w, o.handlerOptions)
	}

	if o.name != "" {
		o.handler = o.handler.WithGroup(o.name)
	}

	return o
}

func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

func WithLevel(level slog.Level) Option {
	return func(o *options) {
		o.level = level
	}
}

func WithOutput(w io.Writer) Option {
	return func(o *options) {
		o.w = w
	}
}

func WithHandler(h slog.Handler) Option {
	return func(o *options) {
		o.handler = h
	}
}

func WithDisabled(disabled bool) Option {
	return func(o *options) {
		o.disabled = disabled
	}
}

func WithTextHandlerOptions(handlerOptions *slog.HandlerOptions) Option {
	return func(o *options) {
		o.handlerOptions = handlerOptions
	}
}
