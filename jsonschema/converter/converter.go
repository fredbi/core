package converter

import (
	"errors"

	"github.com/fredbi/core/jsonschema"
)

// Converter knows how to convert a JSON schema from one version to another.
type Converter struct {
}

func New(opts ...Option) *Converter {
	return &Converter{}
}

// Convert a schema into another JSON schema version.
//
// This may error if some constructs that are not portable to the target version are found.
//
// TODO: examples
func (c *Converter) Convert(s jsonschema.Schema, to jsonschema.SchemaVersion) (jsonschema.Schema, error) {
	return jsonschema.Make(), errors.New("not implemented")
}
