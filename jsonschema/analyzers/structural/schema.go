package structural

import (
	"fmt"
	"iter"

	"github.com/fredbi/core/json/dynamic"
	"github.com/fredbi/core/jsonschema"
	"github.com/fredbi/core/jsonschema/analyzers"
)

type analyzedObject struct {
	// dependencies
	Index         int64 // current index in the dependency graph
	RequiredIndex int64 // -1 if no requirement

	// all the rest is kept private
	id   analyzers.UniqueID // UUID of the package
	meta Metadata

	// naming
	name string
	path string

	pattern string
	format  string

	defaultValue dynamic.JSON
	enum         []dynamic.JSON

	// internal
	dollarID   string // "$id"
	ref        string // $ref path
	isCircular bool

	// audit
	auditTrail
}

func (p analyzedObject) ID() analyzers.UniqueID {
	return p.id
}

func (p analyzedObject) Metadata() Metadata {
	return p.meta
}

func (p analyzedObject) Name() string {
	return p.name
}

func (p analyzedObject) Path() string {
	return p.path
}

type SchemaRelation uint8

const (
	SchemaRelationNone SchemaRelation = iota
	SchemaRelationProperty
	SchemaRelationAdditionalProperty
	SchemaRelationPatternProperty
	SchemaRelationItems
	SchemaRelationTupleItems
	SchemaRelationTupleAdditionalProperty
	SchemaRelationAllOf
	SchemaRelationOneOf
	SchemaRelationAnyOf
	// ...
)

type AnalyzedSchemaContext struct {
	from     *AnalyzedSchema
	to       *AnalyzedSchema
	linkType SchemaRelation
	key      string // linkType characterized by a key: property, patternProperty, ...
	index    int    // linkType characterized by an index: allOf, anyOf, oneOf, tuple
}

// AnalyzedSchema is the outcome of the analysis of a JSON schema.
type AnalyzedSchema struct {
	analyzedObject

	document jsonschema.Schema

	kind         analyzers.SchemaKind
	polymorphism analyzers.PolymorphismKind
	scalarKind   analyzers.ScalarKind

	// dependency graph
	parents  []*AnalyzedSchema // -> AnalyzedSchemaContext
	children []*AnalyzedSchema // -> AnalyzedSchemaContext

	extensions Extensions

	// parent information
	parentProperty string // WRONG
	headParent     *AnalyzedSchema

	// structural validations

	// object-related
	namespace          Namespace // namespace for this schema, e.g. properties in objects
	properties         []*AnalyzedSchema
	implicitProperties []*AnalyzedSchema // TODO: add kind of prop

	// composition-related
	parentAllOf    *AnalyzedSchema
	parentAnyOf    *AnalyzedSchema
	parentOneOf    *AnalyzedSchema
	parentBaseType *AnalyzedSchema

	// extra audit
	refactors []refactoringInfo
	// TODO: other validations (for the moment we don't care)
}

// IsRefactored indicates if the schema has been refactored by the analyzer
func (a AnalyzedSchema) IsRefactored() bool {
	return len(a.refactors) != 0
}

func (a AnalyzedSchema) IsCircular() bool {
	return a.isCircular
}

func (a AnalyzedSchema) HasSchemaID() bool {
	return a.dollarID != ""
}

func (a AnalyzedSchema) SchemaID() string {
	return a.dollarID
}

// Parents yields all parent schemas of a given schema.
//
// The result is empty if the schema is a root schema.
//
// Example:
//
//	  $defs:
//		  A:       # <- {analyzed}
//		    type: object
//		  P1:
//		    type: array
//		    items: # <- parent #1
//		      $ref: #/$defs/A
//		  P2:
//		    type: object
//		    properties:
//		      a:   # <- parent #2
//		        $ref: #/$defs/A
func (a AnalyzedSchema) Parents() iter.Seq[AnalyzedSchema] {
	return func(yield func(AnalyzedSchema) bool) {
		for _, parent := range a.parents {
			value := *parent
			if !yield(value) {
				return
			}
		}
	}
}

// IsRoot indicates if the schema has no parent
func (a AnalyzedSchema) IsRoot() bool {
	return len(a.parents) == 0
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

	return len(a.properties) + len(a.implicitProperties)
}

func (a AnalyzedSchema) HeadParent() AnalyzedSchema {
	if a.headParent == nil {
		panic("don't call head parent when no single root is enforced")
	}

	return *a.headParent
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
	if len(a.parents) == 0 || !a.parents[0].IsObject() { // TODO: this is wrong
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

func (a AnalyzedSchema) Children() iter.Seq[AnalyzedSchema] {
	return func(yield func(AnalyzedSchema) bool) {
		for _, child := range a.children {
			value := *child
			if !yield(value) {
				return
			}
		}
	}
}

func (a AnalyzedSchema) IsObject() bool {
	return a.kind == analyzers.SchemaKindObject
}

func (a AnalyzedSchema) IsNull() bool {
	return a.kind == analyzers.SchemaKindScalar && a.scalarKind == analyzers.ScalarKindNull
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

func (a AnalyzedSchema) IsEnumOnly() bool {
	return false
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

// HasPattern indicates if here is a pattern validation
func (a AnalyzedSchema) HasPattern() bool {
	return false // TODO
}

// Pattern ... a
func (a AnalyzedSchema) Pattern() string {
	return "" // TODO
}

// IsAlwaysInvalid indicate that this schema is never valid.
func (a AnalyzedSchema) IsAlwaysInvalid() bool {
	return false // TODO
}
