package golang

import (
	"path/filepath"

	types "github.com/fredbi/core/genmodels/generator/extra-types"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

type ImportsMap []AliasedImport

// type TargetModels []TargetModel

// type ModelsIndex map[string]TargetModel

type LocationPath string
type LocationIndex map[types.UniqueID]LocationPath

type GenModelOptions struct {
	*GenOptions
	WantsOnlyValidation bool // a model with only validation code
}

type GenSchemaTemplateOptions struct {
	*GenOptions
	NeedsSerializer bool
	NeedsValidation bool
	MarshalMode     MarshalMode
	JSONLibPath     string
	Serializer      SerializerSelector
}

type TargetModel struct {
	GenModelOptions
	ID              types.UniqueID // unique file qualifier, e.g. github.com/fredbi/core/models/model.go
	Name            string         // original name (from the spec, if any), e.g. "model"
	Package         string         // package short name (e.g. "models")
	PackageLocation string         // relative path to the package (e.g. "models/subpackage/enums")
	FullPackage     string         // fully qualified package name (e.g. "github.com/fredbi/core/models")
	File            string         // file name (e.g. model.go)
	StdImports      ImportsMap     // imports from the standard library
	Imports         ImportsMap     // non-standard imports

	Schemas      []TargetSchema   // all the schemas to produce in a single source model file
	Dependencies []types.UniqueID // UniqueID's of models this schema depends on
}

// FileName resolves the relative path to the file name.
func (m TargetModel) FileName() string {
	return filepath.Join(m.PackageLocation, m.File)
}

type TargetSchema struct {
	GenSchemaTemplateOptions
	TypeDefinition
	TypeValidations

	Source *structural.AnalyzedSchema
}

type Metadata struct {
	// s stores.Store // TODO: use this to avoid storing in memory all the comments & metadata text
	ID             types.UniqueID // unique type identifier, e.g. github.com/fredbi/core/models.Model
	Name           string         // original name from the spec
	Title          string         // store.Handle
	Definition     string         // store.Handle
	JSONComment    string         // store.Handle
	Path           string
	Example        any
	Examples       []any
	Annotations    []string                  // reverse-spec annotations
	*OpenAPIMedata                           // nil if schema does not originate from an OAI schema or spec
	Report         []types.InformationReport // generation report: warnings, tracking decisions etc
	Related        []types.UniqueID          // related types (e.g container -> children)
}

type OpenAPIMedata struct {
	Tags        []string
	ExternalDoc ExternalDocumentation
}

type ExternalDocumentation struct {
	Description string
	URL         string
}

type MarshalTemplateOptions struct {
}

type TypeValidations struct {
	HasValidations bool
	ReadOnly       bool
	WriteOnly      bool

	// TODO: embed JSON schema typed validations

	ValidatorInternals
}

type ValidatorInternals struct {
	KeyVar   string
	IndexVar string
}

type ContainerContextFlags struct {
	IsPointer   bool // the contained element is a pointer
	IsAnonymous bool // the contained element is anonymous (for structs, primitive types)
	IsEmbedded  bool // the contained element is an embedded type (for structs, interfaces)
}

type ContainerContext struct {
	ContainerContextFlags
	ReceiverName string
	Schema       *TargetSchema
}

// NamedContainerContext is used for structs and interfaces to name and element
type NamedContainerContext struct {
	ContainerContext

	Name       string // original name for this contained schema, from the spec (e.g. property name)
	Identifier string // go identifier for this contained schema
	StructTags string
	IsExported bool
}

type MethodContainerContext struct {
	Name              string
	Identifier        string // go identifier for this method, e.g. "GetProperty"
	ReceiverName      string
	MethodKind        MethodKindSelector
	IsPointerReceiver bool
	UnderlyingField   *TargetSchema // when the method is a Getter or Setter, we only specify the UnderlyingField, not the arrays below
	Parameters        []NamedContainerContext
	Returns           []ContainerContext
}

type ContainerFlags struct {
	IsType bool // the container require a type declaration, like "type A {GoType}"

	IsStruct    bool
	IsMap       bool
	IsSlice     bool
	IsAny       bool
	IsTuple     bool
	IsStream    bool
	IsPrimitive bool
	IsInterface bool

	IsEnum      bool
	IsExternal  bool
	IsAliased   bool // type A = B
	IsRedefined bool // type A B
	IsElement   bool
	IsGeneric   bool
	IsNullable  bool // should support explicit "null" value vs undefined (this is not the same as IsPointer)

	HasDiscriminator bool
	HasEmbedded      bool // contains an embedded type (for structs, interfaces)
	HasInterface     bool // contains an interface
	HasStream        bool // contains an io.Reader or io.Writer
	HasEnum          bool // has some enum or const validation
}

// TypeDefinition describes a type definition statement.
type TypeDefinition struct {
	Metadata
	Identifier string // type identifier, as in "type A ..."
	GoType     string // type specification, as in "type A string", not applicable to structs or interfaces
	ContainerFlags

	// maps
	Key *ContainerContext

	// maps and slices
	Element *ContainerContext

	// structs & tuples
	Fields []NamedContainerContext

	// interfaces
	Methods []MethodContainerContext // GetX, SetX
	DiscriminatedTypes

	DefaultValue any
}

type DiscriminatedTypes struct {
	IsBaseType         bool
	DiscriminatorField string
	DiscriminatorValue string
}
