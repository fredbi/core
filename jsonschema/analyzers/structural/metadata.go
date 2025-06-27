package structural

import (
	"github.com/fredbi/core/jsonschema"
	"github.com/fredbi/core/jsonschema/analyzers"
)

// AnnotateSchema allows to alter [Metadata] for a schema.
func (a *SchemaAnalyzer) AnnotateSchema(s AnalyzedSchema, meta Metadata) {
	schema, ok := a.schemas.SchemaByID(s.ID())
	if !ok {
		return
	}
	schema.meta = meta // TODO: merge not overwrite => use Metadata builder in jsonschema (replace "structural builder")
}

type Metadata struct {
	ID   analyzers.UniqueID // unique schema identifier, e.g. UUID
	Path string
	jsonschema.Metadata
	tags []string // x-go-tag
}

func (m Metadata) HasTags() bool {
	return len(m.tags) > 0
}

func (m Metadata) Tags() []string {
	return m.tags
}
