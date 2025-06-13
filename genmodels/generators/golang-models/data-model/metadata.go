package model

import (
	types "github.com/fredbi/core/genmodels/generators/extra-types"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/jsonschema/analyzers"
)

// Metadata collected about a schema.
type Metadata struct {
	ID              analyzers.UniqueID // unique schema identifier, e.g. UUID
	OriginalName    string             // original name from the spec
	OpenAPIMetadata *OpenAPIMedata     // nil if schema does not originate from an OAI schema or spec

	title       stores.Handle // store.Handle
	description stores.Handle // store.Handle
	jsonComment stores.Handle // store.Handle
	Path        string
	Examples    []any
	Annotations []string                  // reverse-spec annotations
	Report      []types.InformationReport // generation report: warnings, tracking decisions etc
	Related     []analyzers.UniqueID      // related types (e.g container -> children). Used in doc string.

	s stores.Store // avoids storing in memory all the comments & metadata text
}

func (m Metadata) Title() string {
	v := m.s.Get(m.title)

	return v.String()
}

func (m Metadata) Description() string {
	v := m.s.Get(m.description)

	return v.String()
}

func (m Metadata) JSONComment() string {
	v := m.s.Get(m.jsonComment)

	return v.String()
}

func (m Metadata) NumExamples() int {
	return len(m.Examples)
}

// OpenAPIMedata collects metadata specifically defined by OpenAPI schemas,
// such as summary, externalDocs, xml and tags.
//
// Notice that tags are not directly supported by OpenAPI schemas.
// However, the [structural.Analyzer] may infer tags for schemas when they are used by operations.
type OpenAPIMedata struct {
	Tags             []string
	Summary          string
	ExternalDoc      ExternalDocumentation
	XML              OpenAPIXML
	ExamplesMetadata []OpenAPIExampleMetadata
}

type ExternalDocumentation struct {
	URL string

	description stores.Handle
	s           stores.Store
}

func (e ExternalDocumentation) Description() string {
	v := e.s.Get(e.description)

	return v.String()
}

// OpenAPIXML describes an OpenAPI XML object.
type OpenAPIXML struct {
	Name      string
	Namespace string
	Prefix    string
	Attribute string
	Wrapped   string
}

// OpenAPIExampleMetadata captures metadata about examples, in the order provided in [Metadata.Examples]
//
// TODO: use Store
type OpenAPIExampleMetadata struct {
	Summary       string
	Description   string
	ExternalValue string
}
