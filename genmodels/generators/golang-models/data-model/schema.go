package model

import (
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

// TargetSchema corresponds to a go type definition.
//
// Depending on the [TargetCodeFlags], a single schema definition may trigger a different generation
// (e.g. type definition, validation method, test code...).
type TargetSchema struct {
	GenSchemaTemplateOptions
	TypeDefinition
	TypeValidations

	Index   int64
	Imports ImportsMap // all imports (inherited from the parent model)

	Source *structural.AnalyzedSchema
	// ParentDependencies []analyzers.UniqueID // UniqueID's of models this schema depends on
}

type GenSchemaTemplateOptions struct {
	*GenOptions
	TargetCodeFlags

	NeedsSerializer bool
	MarshalMode     MarshalMode
	JSONLibPath     string
	Serializer      SerializerSelector
}

// TargetCodeFlags instructs the generator about the kind of content to generate.
//
// This may be only a type definition with its methods,
// schema validation code split apart, or test code.
type TargetCodeFlags struct {
	NeedsOnlyValidation bool // a model with only validation code
	NeedsType           bool // the container requires a type declaration, like "type A {GoType}"
	NeedsTest           bool // the container requires test code
	NeedsValidation     bool // the container requires test code
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
	kind TargetKind

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

func (f ContainerFlags) IsStruct() bool    { return f.kind == TargetKindStruct }
func (f ContainerFlags) IsSlice() bool     { return f.kind == TargetKindSlice }
func (f ContainerFlags) IsMap() bool       { return f.kind == TargetKindMap }
func (f ContainerFlags) IsAny() bool       { return f.kind == TargetKindAny }
func (f ContainerFlags) IsTuple() bool     { return f.kind == TargetKindTuple }
func (f ContainerFlags) IsStream() bool    { return f.kind == TargetKindStream }
func (f ContainerFlags) IsPrimitive() bool { return f.kind == TargetKindPrimitive }
func (f ContainerFlags) IsInterface() bool { return f.kind == TargetKindInterface }
func (f ContainerFlags) IsArray() bool     { return f.kind == TargetKindArray }

// IsNilable indicates that the contained object may take the go nil value (this is not the same as the JSON null value).
func (f ContainerFlags) IsNilable() bool {
	return f.IsInterface() || f.IsMap() || f.IsSlice() || f.IsAny() || f.IsStream()
}

type DiscriminatedTypes struct {
	IsBaseType         bool
	DiscriminatorField string
	DiscriminatorValue string
}

/*
type MarshalTemplateOptions struct {
}
*/
