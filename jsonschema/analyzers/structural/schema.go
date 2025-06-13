package structural

import (
	"fmt"

	"github.com/fredbi/core/jsonschema/analyzers"
)

// TODO: how to mock AnalyzedSchema??

type analyzedObject struct {
	ID analyzers.UniqueID // UUID of the package
	// dependencies
	Index         int64 // current index in the dependency graph
	RequiredIndex int64 // -1 if no requirement
	AuditTrail

	//Ref
	RefLocation string   // $ref path
	Tags        []string // x-go-tag

	// naming
	name string
	path string
}

func (p analyzedObject) Name() string {
	return p.name
}

func (p analyzedObject) Path() string {
	return p.path
}

// AnalyzedPackage is the outcome of a package when bundling a JSON schema.
//
// Package hierarchy is a tree, not just a DAG.
type AnalyzedPackage struct {
	analyzedObject

	schemas        []*AnalyzedSchema // schemas defined in this package
	parent         *AnalyzedPackage
	children       []*AnalyzedPackage
	ultimateParent *AnalyzedPackage
}

func (p AnalyzedPackage) IsEmpty() bool {
	return p.ID == ""
}

func (p AnalyzedPackage) Parent() AnalyzedPackage {
	if p.parent != nil {
		return *p.parent
	}

	return AnalyzedPackage{}
}

func (p AnalyzedPackage) Children() []AnalyzedPackage {
	values := make([]AnalyzedPackage, len(p.children))
	for i, child := range p.children {
		values[i] = *child
	}

	return values
}

func (p AnalyzedPackage) Schemas() []AnalyzedSchema {
	values := make([]AnalyzedSchema, len(p.schemas))
	for i, schema := range p.schemas {
		values[i] = *schema
	}

	return values
}

// AnalyzedSchema is the outcome of the analysis of a JSON schema.
type AnalyzedSchema struct {
	analyzedObject
	DollarID string // "$id"

	kind         analyzers.SchemaKind
	polymorphism analyzers.PolymorphismKind
	scalarKind   analyzers.ScalarKind

	// layout

	// other extensions
	extensions Extensions
	namespace  Namespace
	properties []*AnalyzedSchema

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

// NumProperties counts the number of explicitly defined properties
func (a AnalyzedSchema) NumProperties() int {
	if !a.IsObject() {
		return 0
	}

	return len(a.properties)
}

// NumAllProperties counts explicit and implicit properties
func (a AnalyzedSchema) NumAllProperties() int {
	if !a.IsObject() {
		return 0
	}

	return 0
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

// IsImplicitAdditionalProperty indicates if the additional property comes from
// JSON schema defaults (true)
func (a AnalyzedSchema) IsImplicitAdditionalProperty() bool {
	return false // TODO
}

func (a AnalyzedSchema) IsPatternProperty() bool {
	return false // TODO
}

func (a AnalyzedSchema) IsItems() bool {
	return false // TODO
}

func (a AnalyzedSchema) IsSubType() bool {
	return false // TODO
}

func (a AnalyzedSchema) BaseType() AnalyzedSchema {
	if !a.IsSubType() {
		panic("yay")
	}
	return AnalyzedSchema{} // TODO
}

func (a AnalyzedSchema) PatternPropertyIndex() int {
	return 0 // TODO
}

func (a AnalyzedSchema) ParentProperty() string {
	if len(a.parents) != 1 || !a.parents[0].IsObject() {
		return ""
	}

	return a.parentProperty
}

func (a AnalyzedSchema) IsAllOfMember() bool {
	return false // TODO
}

func (a AnalyzedSchema) AllOfMemberIndex() int {
	return 0 // TODO
}

func (a AnalyzedSchema) IsOneOfMember() bool {
	return false // TODO
}

func (a AnalyzedSchema) OneOfMemberIndex() int {
	return 0 // TODO
}

func (a AnalyzedSchema) IsAnyOfMember() bool {
	return false // TODO
}

func (a AnalyzedSchema) AnyOfMemberIndex() int {
	return 0 // TODO
}

func (a AnalyzedSchema) IsTupleMember() bool {
	return false // TODO
}

func (a AnalyzedSchema) TupleMemberIndex() int {
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

func (a AnalyzedSchema) ScalarKind() analyzers.ScalarKind {
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
	return a.name == ""
}

func (a AnalyzedSchema) WasAnonymous() bool {
	return a.OriginalName() == ""
}

func (a AnalyzedSchema) IsNamed() bool {
	return a.name != ""
}

func (a AnalyzedSchema) IsRoot() bool {
	return len(a.parents) == 0
}

func (a AnalyzedSchema) HasSingleParent() bool {
	return len(a.parents) == 1
}

func (a AnalyzedSchema) HasEnum() bool {
	return false // TODO
}

func (a AnalyzedSchema) HasFormatValidation() bool {
	return false // TODO
}

func (a AnalyzedSchema) FormatValidation() string {
	return "" // TODO
}

// IsAlwaysInvalid indicate that this schema is never valid.
func (a AnalyzedSchema) IsAlwaysInvalid() bool {
	return false // TODO
}
