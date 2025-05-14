package differ

type ReportOption func(*reportOptions)

type reportOptions struct {
}

func WithThreshold(level Severity) Option {
	return func(o *options) {
	}
}
