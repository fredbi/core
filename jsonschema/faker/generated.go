package faker

import (
	"github.com/fredbi/core/json"
	"github.com/fredbi/core/jsonschema"
)

// Generated outcome of the [SchemaFaker] or the [DataFaker]
type Generated struct {
	kind   generatedKind
	schema jsonschema.Schema
	doc    json.Document
	valid  bool
}

func (g Generated) ShouldBeValid() bool {
	return g.valid
}

func (g Generated) Schema() jsonschema.Schema {
	return g.schema
}

func (g Generated) Document() json.Document {
	switch g.kind {
	case generatedKindSchema:
		return g.schema.Document
	case generatedKindData:
		return g.doc
	default:
		panic("invalid generated kind")
	}
}

type generatedKind uint8

const (
	generatedKindSchema generatedKind = iota + 1
	generatedKindData
)
