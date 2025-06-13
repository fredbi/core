package model

import (
	settings "github.com/fredbi/core/genmodels/generators/common-settings"
)

/*
  other bright ideas, but there is already a lot to do:
  * several YAML libraries are now available (go-ccy has one)
  * use string interning for object keys
  * polymorphic inheritance may generate an interface or a concrete type
  * regexp-based validations to support ECMA regexpes
  * MarshalContent for "simple" types (parameters, headers)?
	* map object as map rather than struct
*/

/*
TODO: support other tags such as x-go-type, x-go-external, x-go-omitempty etc
*/

type AliasedImport struct {
	Alias   string
	Package string
}

// GenOptions lists all available code generation options for the golang model target.
type GenOptions struct {
	// targets selection
	GenTargetOptions `section:"target"`

	// customization options
	GenCustomOptions `section:"customization"`

	// serializer options
	GenSerializerOptions `section:"serializer"`

	// validations options
	GenValidationOptions `section:"validation"`

	GenSchemaOptions `section:"schema"`
}

// GenTargetOptions defines top-level generation options.
type GenTargetOptions struct {
	settings.GenTargetOptions `section:",squash"`

	WantsGoGenerate bool
	WantsGoMod      bool
	WantsPkgDoc     bool   // generate a doc.go file for each package
	WantsPkgReadme  bool   // generate a README.md file for each package
	MinGoVersion    string // when WantsGoMod is true, generate with a required go version
	ImportOptions
}

func (o GenTargetOptions) WantsPkgArtifact() bool {
	return o.WantsPkgDoc || o.WantsPkgReadme
}

type ImportOptions struct {
	ImportRuntime string
	ImportStrFmt  string
}

type GenSchemaOptions struct {
	settings.GenSchemaOptions `section:",squash"`

	WantsExternalType bool
	ExternalType      GenExternalTypeOptions

	GenNumberOptions `section:"number"`
	GenObjectOptions `section:"object"`
	GenArrayOptions  `section:"array"`
}

type GenExternalTypeOptions struct {
	Type                  string
	Import                AliasedImport
	WantsExternalEmbedded bool `aliases:"embedded"`
}

// GenCustomOptions covers options to use templates in a customized way
type GenCustomOptions struct {
	settings.GenCustomOptions `section:",squash"`
}

// GenLayoutOptions describes top-level layout options.
type GenLayoutOptions struct {
	settings.GenLayoutOptions `section:",squash"`
}

type GenValidationOptions struct {
	settings.GenValidationOptions `section:",squash"`
}

type GenSerializerOptions struct {
	settings.GenSerializerOptions `section:",squash"`

	WantsPool       bool
	MarshalMode     MarshalMode // select which serialization formats should be supported
	JSONLibSelector JSONLibSelector
}

type GenNumberOptions struct {
	IntegerMappingSelector IntegerMappingSelector
	DefaultIntegerGoType   string // defaults to int64
	DecimalMappingSelector DecimalMappingSelector
	DefaultDecimalGoType   string // defaut to float64
}

type GenObjectOptions struct {
	WantsStructTags  bool
	WantsXMLTags     bool
	WantsObjectAsMap bool     // the default is to construct JSON objects as structs, but this may force it to a map instead
	WantsOmitEmpty   bool     `aliases:"omitempty"`
	MapType          string   // default is "map", but can be any generic type that behaves as a map (i.e. iter.Seq2)
	ExtraStructTags  []string `aliases:"customTag"`
}

type GenArrayOptions struct {
	SliceType string // default is "[]", but can be any generic type that behaves as a slice (i.e. iter.Seq2<int,T>)
}
