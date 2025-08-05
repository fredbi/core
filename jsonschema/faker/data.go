package faker

import (
	"iter"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/jsonschema"
)

// DataFaker generates random JSON data based on a json schema.
type DataFaker struct {
	*dataOptions
	schema jsonschema.Schema
}

func NewDataFaker(schema jsonschema.Schema, opts ...DataOption) DataFaker {
	return DataFaker{}
}

func (f DataFaker) Generate() Generated {
	return Generated{kind: generatedKindData, schema: f.schema, doc: json.Make()} // TODO
}

func (f DataFaker) GenerateMany(n int) iter.Seq[Generated] {
	return nil // TODO
}
