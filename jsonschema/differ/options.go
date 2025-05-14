package differ

type Option func(*options)

type options struct {
}

func WithIgnoreCosmeticChanges(enabled bool) Option {
	return func(o *options) {
	}
}
