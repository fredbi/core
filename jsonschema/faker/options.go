package faker

import "github.com/fredbi/core/jsonschema"

type Option func(*options)

type DistorsionLevel uint8
type ComplexityLevel uint8

type options struct{}

func WithSeed(seed int64) Option {
	return func(o *options) {
	}
}

func WithOnlyValid(enabled bool) Option {
	return func(o *options) {
	}
}

func WithOnlyInvalid(enabled bool, garbling DistorsionLevel) Option {
	return func(o *options) {
	}
}

func WithComplexity(c ComplexityLevel) Option {
	return func(o *options) {
	}
}

func WithVersion(c jsonschema.SchemaVersion) Option {
	return func(o *options) {
	}
}

func UseAllOf(enabled bool) Option {
	return func(o *options) {
	}
}

func UseAnyOf(enabled bool) Option {
	return func(o *options) {
	}
}

func UseOneOf(enabled bool) Option {
	return func(o *options) {
	}
}

func UseAdditionalItems(enabled bool) Option {
	return func(o *options) {
	}
}

func UseAdditionalProperties(enabled bool) Option {
	return func(o *options) {
	}
}

func UsePattern(enabled bool) Option {
	return func(o *options) {
	}
}
