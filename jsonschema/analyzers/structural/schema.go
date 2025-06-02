package structural

import (
	"fmt"

	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/jsonschema/analyzers"
)

// AnalyzedSchema is the outcome of the analysis on a JSON schema.
type AnalyzedSchema struct {
	ID analyzers.UniqueID // UUID of the schema
	// dependencies
	Index         int64
	RequiredIndex int64  // -1 if no requirement
	DollarID      string // "$id"
	kind          analyzers.SchemaKind
	polymorphism  analyzers.PolymorphismKind
	scalarKind    token.Kind

	// layout
	//Ref
	RefLocation string   // $ref path
	Tags        []string // x-go-tag

	// naming
	Name string
	Path string

	// other extensions
	extensions Extensions
	namespace  Namespace
	properties []*AnalyzedSchema

	audit          AuditTrail
	parents        []*AnalyzedSchema
	children       []*AnalyzedSchema
	parentProperty string
	ultimateParent *AnalyzedSchema
}

func (a AnalyzedSchema) Parents() []AnalyzedSchema {
	values := make([]AnalyzedSchema, len(a.parents))
	for i, parent := range a.parents {
		values[i] = *parent
	}

	return values
}

func (a AnalyzedSchema) NumProperties() int {
	if !a.IsObject() {
		return 0
	}

	return len(a.properties)
}

func (a AnalyzedSchema) UltimateParent() AnalyzedSchema {
	if a.ultimateParent == nil {
		panic("don't call ultimate parent when no single root is enforced")
	}
	return *a.ultimateParent
}

func (a AnalyzedSchema) Parent() AnalyzedSchema {
	if a.IsRoot() {
		panic("don't call Parent() when AnalyzedSchema is root")
	}

	if !a.IsAnonymous() {
		if !a.HasSingleParent() {
			panic("don't call Parent() when parents are multiple")
		}

		return *a.parents[0]
	}

	if !a.HasSingleParent() {
		panic(fmt.Errorf("internal error: inconsistent non-root anonymous schema with multiple parents: %v", a))
	}

	return *a.parents[0]
}

// HasParentProperty indicates if the schema is referred to by a parent object property.
//
// The parent property may be defined by the "properties" of the parent, or just by "required", "dependentSchema"
// or "dependentRequired", or "if-then-else" statements (implicit property).
//
// TODO: patternProperties
func (a AnalyzedSchema) HasParentProperty() bool {
	if len(a.parents) != 1 || !a.parents[0].IsObject() {
		return false
	}

	return true
}

func (a AnalyzedSchema) IsAdditionalProperty() bool {
	return false // TODO
}

func (a AnalyzedSchema) ParentProperty() string {
	if len(a.parents) != 1 || !a.parents[0].IsObject() {
		return ""
	}

	return a.parentProperty
}

func (a AnalyzedSchema) IsAllOfMember() int {
	return 0 // TODO
}

func (a AnalyzedSchema) IsOneOfMember() int {
	return 0 // TODO
}

func (a AnalyzedSchema) IsAnyOfMember() int {
	return 0 // TODO
}

func (a AnalyzedSchema) IsTupleMember() int {
	return 0 // TODO
}

// IsTupleAdditionalItems indicates if a schema is located in the additionalItems (or items) section of a tuple schema.
func (a AnalyzedSchema) IsTupleAdditionalItems() bool {
	return false // TODO
}

func (a AnalyzedSchema) Children() []AnalyzedSchema {
	values := make([]AnalyzedSchema, len(a.children))
	for i, child := range a.children {
		values[i] = *child
	}

	return values
}

func (a AnalyzedSchema) IsObject() bool {
	return a.kind == analyzers.SchemaKindObject
}

func (a AnalyzedSchema) IsArray() bool {
	return a.kind == analyzers.SchemaKindArray
}

func (a AnalyzedSchema) IsTuple() bool {
	return a.kind == analyzers.SchemaKindTuple
}

func (a AnalyzedSchema) IsScalar() bool {
	return a.kind == analyzers.SchemaKindScalar
}

func (a AnalyzedSchema) ScalarKind() token.Kind {
	return a.scalarKind
}

func (a AnalyzedSchema) IsPolymorphic() bool {
	return a.kind == analyzers.SchemaKindPolymorphic
}

func (a AnalyzedSchema) IsAnyWithoutValidation() bool {
	return a.kind == analyzers.SchemaKindNone
}

// IsEnum is a schema that boils down (after reduction) to a const or enum.
func (a AnalyzedSchema) IsEnum() bool {
	return false // TODO
}

func (a AnalyzedSchema) Extensions() Extensions {
	return a.extensions
}

func (a AnalyzedSchema) GetExtension(extension string, aliases ...string) (any, bool) {
	return a.extensions.Get(extension, aliases...)
}

func (a AnalyzedSchema) HasExtension(extension string, aliases ...string) bool {
	return a.extensions.Has(extension, aliases...)
}

func (a AnalyzedSchema) IsAnonymous() bool {
	return a.Name == ""
}

func (a AnalyzedSchema) WasAnonymous() bool {
	return a.audit.OriginalName == ""
}

func (a AnalyzedSchema) IsNamed() bool {
	return a.Name != ""
}

func (a AnalyzedSchema) IsRoot() bool {
	return len(a.parents) == 0
}

func (a AnalyzedSchema) HasSingleParent() bool {
	return len(a.parents) == 1
}

func (a AnalyzedSchema) HasNameOverride() bool {
	return a.audit.NameOverride != ""
}
