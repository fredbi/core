package faker

import (
	schemafaker "github.com/fredbi/core/jsonschema/faker"
	"github.com/fredbi/core/spec"
)

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

func WithVersion(c spec.OpenAPIVersion) Option {
	return func(o *options) {
	}
}

func UseWebHooks(enabled bool) Option {
	return func(o *options) {
	}
}

func UseSecurity(enabled bool) Option {
	return func(o *options) {
	}
}

func UseNullable(enabled bool) Option {
	return func(o *options) {
	}
}

func WithJSONSchemaOptions(opts ...schemafaker.Option) Option {
	return func(o *options) {
	}
}

func UseOneOf(enabled bool) Option {
	return func(o *options) {
	}
}

func UseAllOf(enabled bool) Option {
	return func(o *options) {
	}
}

func UseRef(enabled bool) Option {
	return func(o *options) {
	}
}

func UseAdvancedRef(enabled bool) Option {
	return func(o *options) {
	}
}
