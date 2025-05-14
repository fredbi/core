package writer

type Option func(*options)

type options struct {
	nilMapAsEmpty   bool
	nilSliceAsEmpty bool
	noEscapeHTML    bool
}

var defaultOptions = options{}

func optionsWithDefaults(opts []Option) options {
	if len(opts) == 0 {
		return defaultOptions
	}

	o := options{}

	for _, apply := range opts {
		apply(&o)
	}

	return o
}
