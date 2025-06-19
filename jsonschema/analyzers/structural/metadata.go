package structural

import (
	"github.com/fredbi/core/json/dynamic"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/jsonschema/analyzers"
)

// AnnotateSchema allows to alter [Metadata] for a schema.
func (a *SchemaAnalyzer) AnnotateSchema(s AnalyzedSchema, meta Metadata) {
	schema, ok := a.index[s.ID()]
	if !ok {
		return
	}
	schema.meta = meta // TODO: merge not overwrite
}

type Metadata struct {
	ID              analyzers.UniqueID // unique schema identifier, e.g. UUID
	Path            string
	OpenAPIMetadata *OpenAPIMetadata // nil if schema does not originate from an OAI schema or spec
	Examples        []dynamic.JSON

	tags        []string      // x-go-tag
	title       stores.Handle // store.Handle
	description stores.Handle // store.Handle
	jsonComment stores.Handle // store.Handle

	s stores.Store // avoids storing in memory all the comments & metadata text
}

func (m Metadata) Store() stores.Store {
	return m.s
}

func (m Metadata) HasTags() bool {
	return len(m.tags) > 0
}

func (m Metadata) Tags() []string {
	return m.tags
}

func (m Metadata) HasTitle() bool {
	return m.title != stores.HandleZero
}

func (m Metadata) Title() string {
	v := m.s.Get(m.title)

	return v.String()
}

func (m Metadata) HasDescription() bool {
	return m.description != stores.HandleZero
}

func (m Metadata) Description() string {
	v := m.s.Get(m.description)

	return v.String()
}

func (m Metadata) JSONComment() string {
	v := m.s.Get(m.jsonComment)

	return v.String()
}

func (m Metadata) HasExamples() bool {
	return len(m.Examples) > 0
}

func (m Metadata) NumExamples() int {
	return len(m.Examples)
}

// OpenAPIMetadata collects metadata specifically defined by OpenAPI schemas,
// such as summary, externalDocs, xml and tags.
//
// Notice that tags are not directly supported by OpenAPI schemas.
// However, the [structural.Analyzer] may infer tags for schemas when they are used by operations.
type OpenAPIMetadata struct {
	ExamplesMetadata []OpenAPIExampleMetadata
	ExternalDoc      ExternalDocumentation
	XML              OpenAPIXML

	tags    []string
	summary stores.Handle
	s       stores.Store
}

func (o OpenAPIMetadata) Summary() string {
	v := o.s.Get(o.summary)

	return v.String()
}

func (o OpenAPIMetadata) HasTags() bool {
	return len(o.tags) > 0
}

func (o OpenAPIMetadata) Tags() []string {
	return o.tags
}

type ExternalDocumentation struct {
	url         stores.Handle
	description stores.Handle
	s           stores.Store
}

func (e ExternalDocumentation) URL() string {
	v := e.s.Get(e.url)

	return v.String()
}

func (e ExternalDocumentation) HasDescription() bool {
	return e.description != stores.HandleZero
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
type OpenAPIExampleMetadata struct {
	summary       stores.Handle
	description   stores.Handle
	externalValue stores.Handle

	s stores.Store
}

func (e OpenAPIExampleMetadata) HasDescription() bool {
	return e.description != stores.HandleZero
}

func (e OpenAPIExampleMetadata) Description() string {
	v := e.s.Get(e.description)

	return v.String()
}

func (e OpenAPIExampleMetadata) HasSummary() bool {
	return e.summary != stores.HandleZero
}

func (e OpenAPIExampleMetadata) Summary() string {
	v := e.s.Get(e.summary)

	return v.String()
}

func (e OpenAPIExampleMetadata) HasExternalValue() bool {
	return e.externalValue != stores.HandleZero
}

func (e OpenAPIExampleMetadata) ExternalValue() string {
	v := e.s.Get(e.externalValue)

	return v.String()
}
