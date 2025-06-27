package jsonschema

import (
	"iter"
	"slices"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/stores"
)

type Metadata struct {
	s           stores.Store    // avoids storing in memory all the comments & metadata text
	openAPIMeta OpenAPIMetadata // nil if schema does not originate from an OAI schema or spec
	examples    []json.Document
	title       stores.Handle // store.Handle
	description stores.Handle // store.Handle
	jsonComment stores.Handle // store.Handle
	defined     bool
}

func (m Metadata) Store() stores.Store {
	return m.s
}

func (m Metadata) IsDefined() bool {
	return m.defined
}

func (m Metadata) HasExamples() bool {
	return len(m.examples) > 0
}

func (m Metadata) Examples() iter.Seq[json.Document] {
	return slices.Values(m.examples)
}

func (m Metadata) NumExamples() int {
	return len(m.examples)
}

func (m Metadata) HasOpenAPIMetadata() bool {
	return m.openAPIMeta.IsDefined()
}

func (m Metadata) OpenAPIMetadata() OpenAPIMetadata {
	return m.openAPIMeta
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

func (m Metadata) HasJSONComment() bool {
	return m.jsonComment != stores.HandleZero
}

func (m Metadata) JSONComment() string {
	v := m.s.Get(m.jsonComment)

	return v.String()
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
	defined          bool

	tags    []string
	summary stores.Handle
	s       stores.Store
}

func (o OpenAPIMetadata) IsDefined() bool {
	return o.defined
}

func (o OpenAPIMetadata) HasSummary() bool {
	return o.summary != stores.HandleZero
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
	s           stores.Store
	url         stores.Handle
	description stores.Handle
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
	Name      string // TODO: use private fields
	Namespace string
	Prefix    string
	Attribute string
	Wrapped   string
	defined   bool
}

// OpenAPIExampleMetadata captures metadata about examples, in the order provided in [Metadata.Examples]
type OpenAPIExampleMetadata struct {
	s             stores.Store
	summary       stores.Handle
	description   stores.Handle
	externalValue stores.Handle
	defined       bool
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
