package jsonschema

import (
	"iter"
	"slices"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/nodes/light"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/jsonschema/meta"
)

// Metadata holds the parts of the schema core keywords that are only descriptive and do not affect
// either the validation or the structural analysis.
//
// This includes the following JSON schema keywords (including support for OpenAPI dialects):
//
//   - $comment (>= draft 7)
//   - description
//   - title
//   - examples (>= draft 6)
//   - $vocabulary
//   - default
//   - readOnly (>= draft 7)
//   - writeOnly (>= draft 7)
//
// For OpenAPI schemas:
//   - externalDocs
//   - xml
//   - example (OpenAPI v2)
type Metadata struct {
	s            stores.Store         // avoids storing in memory all the comments & metadata text
	openAPIMeta  meta.OpenAPIMetadata // nil if schema does not originate from an OAI schema or spec
	examples     []json.Document
	title        stores.Handle
	description  stores.Handle
	jsonComment  stores.Handle
	defaultValue json.Document
	defined      bool
	vocabulary   json.Document
}

var (
	commentKey      = values.MakeInternedKey("$comment")
	descriptionKey  = values.MakeInternedKey("description")
	titleKey        = values.MakeInternedKey("title")
	examplesKey     = values.MakeInternedKey("examples")
	exampleKey      = values.MakeInternedKey("example")
	vocabularyKey   = values.MakeInternedKey("$vocabulary")
	defaultKey      = values.MakeInternedKey("default")
	readOnlyKey     = values.MakeInternedKey("readOnly")
	writeOnlyKey    = values.MakeInternedKey("writeOnly")
	externalDocsKey = values.MakeInternedKey("externalDocs")
	xmlKey          = values.MakeInternedKey("xml")

	metadataKeys = map[values.InternedKey]struct{}{
		commentKey:      {},
		descriptionKey:  {},
		titleKey:        {},
		examplesKey:     {},
		exampleKey:      {},
		vocabularyKey:   {},
		defaultKey:      {},
		readOnlyKey:     {},
		writeOnlyKey:    {},
		externalDocsKey: {},
		xmlKey:          {},
	}

	metadataConstraints = map[values.InternedKey]VersionRequirements{
		examplesKey:     {MinVersion: VersionDraft6},
		commentKey:      {MinVersion: VersionDraft7},
		readOnlyKey:     {MinVersion: VersionDraft7, MinOAIVersion: VersionOpenAPIv2},
		writeOnlyKey:    {MinVersion: VersionDraft7}, // TODO: check OAI version
		vocabularyKey:   {MinVersion: VersionDraft2019},
		exampleKey:      {MinOAIVersion: VersionOpenAPIv2},
		externalDocsKey: {MinOAIVersion: VersionOpenAPIv2},
		xmlKey:          {MinOAIVersion: VersionOpenAPIv2},
	}
)

func (m Metadata) Store() stores.Store {
	return m.s
}

func (m Metadata) IsDefined() bool {
	return m.defined
}

func (m Metadata) HasDefaultValue() bool {
	return false
}

func (m Metadata) DefaultValue() json.Document {
	return json.EmptyDocument
}

func (m Metadata) Vocabulary() json.Document {
	return json.EmptyDocument
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

func (m Metadata) OpenAPIMetadata() meta.OpenAPIMetadata {
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

func (m *Metadata) decode(ctx *light.ParentContext, key values.InternedKey, vr *VersionRequirements) error {
	return nil
}
