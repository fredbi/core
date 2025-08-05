// TODO: move to parent package. Too complicated with private stuff
package meta

import (
	"github.com/fredbi/core/json/stores"
)

// OpenAPIMetadata collects metadata specifically defined by OpenAPI schemas,
// such as summary, externalDocs, xml and tags.
//
// Notice that tags are not directly supported by OpenAPI schemas.
// However, the [structural.Analyzer] may infer tags for schemas when they are used by operations.
type OpenAPIMetadata struct {
	ExamplesMetadata []OpenAPIExampleMetadata
	ExternalDocs     ExternalDocumentation
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
