package converter

type Option func(*options)

type options struct {
}

func WithSkipIncompatible(enabled bool) Option {
	return func(o *options) {
	}
}

func WithContinueOnError(enabled bool) Option {
	return func(o *options) {
	}
}
