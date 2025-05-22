package settings

import (
	"github.com/fredbi/core/jsonschema"
)

// GenOptions describe common code generation options.
//
// In general, these options should not be language-specific.
//
// All options can be set globally or as "x-***" extensions in the JSON schema.
type GenOptions struct {
	// targets selection
	GenTargetOptions `section:"target"`

	// customization options
	GenCustomOptions `section:"customization"`

	// serializer options
	GenSerializerOptions `section:"serializer"`

	// validations options
	GenValidationOptions `section:"validation"`

	// schema generation options
	GenSchemaOptions `section:"schema"`
}

// GenTargetOptions defines top-level generation options.
type GenTargetOptions struct {
	Copyright       string // a Copyright header to add to all generated source files
	WantsDumpSchema bool   // a dump of the transformed schema used by code gen

	GenJSONSchemaOptions `section:"jsonschema"`
	GenOAISchemaOptions  `section:"openapi-schema"`
	GenLayoutOptions     `section:"code-layout"`
}

type GenSchemaOptions struct {
	WantsExtraMethod bool
	ExtraMethodsMode ExtraMethodsMode
}

type GenJSONSchemaOptions struct {
	DefaultJSONSchemaVersion          jsonschema.Version // the default dialect to use
	WantsJSONSchemaVersion            jsonschema.Version
	WantsEnumConstant                 bool   // generate constants for enum values
	WantsImplicitAdditionalProperties bool   // when true, JSON objects keep non-explicitly defined additionalProperties
	WantsImplicitAdditionalItems      bool   // when true, JSON tuples keep non-explicitly defined additionalItems
	WantsName                         bool   `aliases:"name"`
	ImplyNullIsUndefined              bool   // when true, null and undefined have the same semantics
	FormatsImportPath                 string // fully qualified import path for format types
}

// GenOAISchemaOptions defines extra settings that are specific to the OpenAPI definition of a schema.
type GenOAISchemaOptions struct {
	AllowAdditionalItems  bool // was forbidden in OAIv2, may override this (applies to prefixItems)
	AllowOneOf            bool // was forbidden in OAIv2, may override this
	AllowAnyOf            bool // was forbidden in OAIv2, may override this
	AllowDiscriminator    bool // when false, use only validation to discriminate objects
	AllowInheritance      bool
	AllowNullable         bool // was forbidden in OAIv2, may override this to support x-nullable
	ExamplesMustValidate  bool // true in OAI, not required by JSON schema
	AllowExamplesMetadata bool // OAI example object has metadata
	AllowXML              bool
	AllowRequiredReadOnly bool // was forbidden in OAIv2
	AllowRequiredDefault  bool // was forbidden in OAIv2
	AllowAnnotateEnums    bool // OAIv3 idiom
}

type GenMetadataOptions struct {
	WantsDocString           bool // generate docstring comments for types
	WantsValidationDocString bool
	WantsRelated             bool
	WantsCodegenReport       bool // generates extra comments to explain codegen decisions
	WantsAnnotations         bool // generate extra comments for swagger generate spec
	WantsOAIExternalDocs     bool // include ExternalDocs attribute (OAI) as comments
	WantsOAItags             bool // include Tags attribute (OAI) as comments
	WantsJSONSchemaComment   bool
}

// GenCustomOptions covers options to use templates in a customized way
type GenCustomOptions struct {
	WantsDumpTemplates    bool
	DumpTemplateFormat    string
	BaseTemplatePath      string
	AlternateTemplatePath string
	AllowTemplateOverride bool // allow template overlay
	SkipFmt               bool // skip source formatting (e.g. gofmt for golang target)
	SkipCheckImport       bool // skip import checking (e.g. goimports for golang target)
}

// GenLayoutOptions describes top-level layout options.
type GenLayoutOptions struct {
	TargetImportPath     string                      // the base generation target path
	PackageLayout        PackageLayoutSelector       // options to plan the layout of generated packages
	PackageLayoutOptions PackageLayoutOptionSelector // additional options for package layout
	ModelLayout          ModelLayoutSelector
}

type GenValidationOptions struct {
	WantsValidations     bool // when false, generated types have no standalone validation
	WantFormatValidation bool // JSONSchema spec defines those as "annotations" only
	WantsStringEnumsCI   bool `aliases:"enum-ci"` // when true string enums are case-insensitive
	WantsStrictOutput    bool // when true, validation errors abide by https://json-schema.org/draft/{version}/output/schema
	WantsVerboseOutput   bool // when true, validation errors are verbose
	ValidationLayout     ValidationLayoutSelector
}

type GenSerializerOptions struct {
	WantsSerializer        bool // when false, generated types have to serializer
	WantsMarshal           bool // generates MarshalXXX methods
	WantsUnmarshal         bool // generates UnmarshalXXX methods
	WantsDefaultSerialized bool // if true, default values are serialized - if false, there left undefined, unless required
}
