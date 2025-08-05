package faker

import (
	"github.com/fredbi/core/jsonschema"
	"github.com/fredbi/core/stubs"
)

type SchemaOption func(*schemaOptions)

type DistorsionLevel uint8
type ComplexityLevel uint8

type commonOptions struct {
	seed        int64
	onlyValid   bool
	onlyInvalid bool
	baseFaker   stubs.Faker
	garbling    DistorsionLevel
	complexity  ComplexityLevel
}

type schemaOptions struct {
	commonOptions
}

func WithSchemaSeed(seed int64) SchemaOption {
	return func(o *schemaOptions) {
		o.seed = seed
	}
}

func WithSchemaBaseFaker(faker stubs.Faker) SchemaOption {
	return func(o *schemaOptions) {
		o.baseFaker = faker
	}
}

func WithSchemaOnlyValid(enabled bool) SchemaOption {
	return func(o *schemaOptions) {
	}
}

func WithSchemaOnlyInvalid(enabled bool, garbling DistorsionLevel) SchemaOption {
	return func(o *schemaOptions) {
	}
}

func WithSchemaComplexity(c ComplexityLevel) SchemaOption {
	return func(o *schemaOptions) {
	}
}

func WithSchemaVersion(c jsonschema.Version) SchemaOption {
	return func(o *schemaOptions) {
	}
}

func UseAllOf(enabled bool) SchemaOption {
	return func(o *schemaOptions) {
	}
}

func UseAnyOf(enabled bool) SchemaOption {
	return func(o *schemaOptions) {
	}
}

func UseOneOf(enabled bool) SchemaOption {
	return func(o *schemaOptions) {
	}
}

func UseAdditionalItems(enabled bool) SchemaOption {
	return func(o *schemaOptions) {
	}
}

func UseAdditionalProperties(enabled bool) SchemaOption {
	return func(o *schemaOptions) {
	}
}

func UsePattern(enabled bool) SchemaOption {
	return func(o *schemaOptions) {
	}
}

func UsePatternProperties(enabled bool) SchemaOption {
	return func(o *schemaOptions) {
	}
}

func UseUnevaluatedProperties(enabled bool) SchemaOption {
	return func(o *schemaOptions) {
	}
}

func UseRef(enabled bool /* TODO: other $ref-specific schemaOptions*/) SchemaOption {
	return func(o *schemaOptions) {
	}
}
