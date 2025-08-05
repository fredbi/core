package faker

import (
	"github.com/fredbi/core/stubs"
)

type DataOption func(*dataOptions)

type dataOptions struct {
	commonOptions

	allowInvalidJSON bool
}

func WithDataSeed(seed int64) DataOption {
	return func(o *dataOptions) {
		o.seed = seed
	}
}

func WithDataBaseFaker(faker stubs.Faker) SchemaOption {
	return func(o *schemaOptions) {
		o.baseFaker = faker
	}
}

func WithDataOnlyValid(enabled bool) DataOption {
	return func(o *dataOptions) {
	}
}

func WithDataOnlyInvalid(enabled bool, garbling DistorsionLevel) DataOption {
	return func(o *dataOptions) {
	}
}

func WithDataInvalidJSON(enabled bool, garbling DistorsionLevel) DataOption {
	return func(o *dataOptions) {
		o.allowInvalidJSON = enabled
	}
}

func WithDataComplexity(c ComplexityLevel) DataOption {
	return func(o *dataOptions) {
	}
}
