package faker

import (
	"iter"

	"github.com/fredbi/core/jsonschema"
)

// SchemaFaker generates random JSON schemas.
type SchemaFaker struct {
	*schemaOptions
}

func NewSchemaFaker(opts ...SchemaOption) SchemaFaker {
	return SchemaFaker{}
}

func (f SchemaFaker) Generate() Generated {
	return Generated{kind: generatedKindSchema, schema: jsonschema.Make()} // TODO
}

func (f SchemaFaker) GenerateMany(n int) iter.Seq[Generated] {
	return nil // TODO
}
