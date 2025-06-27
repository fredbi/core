package structural

import (
	"fmt"

	"github.com/fredbi/core/json/dynamic"
	"github.com/fredbi/core/jsonschema/analyzers"
)

// SchemaBuilder builds an [AnalyzedSchema].
//
// This is used internally by the [SchemaAnalyzer].
// It may also be used by consumers of [AnalyzedSchema] s to build mocks.
type SchemaBuilder struct {
	err error
	s   AnalyzedSchema
}

func MakeSchemaBuilder(SchemaBuilder) SchemaBuilder {
	return SchemaBuilder{}
}

func (b SchemaBuilder) Schema() AnalyzedSchema {
	if b.err == nil {
		return b.s
	}

	return AnalyzedSchema{}
}

func (b SchemaBuilder) WithID(id analyzers.UniqueID) SchemaBuilder {
	b.s.id = id

	return b
}

func (b SchemaBuilder) WithMetadata(meta Metadata) SchemaBuilder {
	meta.ID = b.s.id
	b.s.meta = meta

	return b
}

func (b SchemaBuilder) WithName(name string) SchemaBuilder {
	b.s.name = name

	return b
}

func (b SchemaBuilder) WithPath(path string) SchemaBuilder {
	b.s.path = path

	return b
}

func (b SchemaBuilder) WithIsRefactored(enabled bool) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithIsCircular(enabled bool) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithSchemaID(id string) SchemaBuilder {
	b.s.dollarID = id

	return b
}

func (b SchemaBuilder) WithParents(parents ...AnalyzedSchema) SchemaBuilder {
	for _, p := range parents {
		parent := p
		b.s.parents = append(b.s.parents, &parent)
	}

	return b
}

func (b SchemaBuilder) WithProperties(properties ...AnalyzedSchema) SchemaBuilder {
	if !b.s.IsObject() {
		b.err = fmt.Errorf("WithProperties applies to object schemas only: %w", ErrSchemaBuilder)

		return b
	}

	for _, p := range properties {
		property := p
		b.s.properties = append(b.s.properties, &property)
	}

	return b
}

func (b SchemaBuilder) WithHeadParent(head AnalyzedSchema) SchemaBuilder {
	h := head
	b.s.headParent = &h

	return b
}

func (b SchemaBuilder) WithHasParentProperty(bool) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithAdditionalProperty(additional AnalyzedSchema) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithIsImplicitAdditionalProperty(bool) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithIsPatternProperty(bool) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithIsItems(bool) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithIsSubType(bool) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithBaseType(AnalyzedSchema) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithPatternPropertyIndex(int) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithParentProperty(string) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithIsAllOfMember(bool) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithAllOfMemberIndex(int) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithIsOneOfMember(bool) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithOneOfMemberIndex(int) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithIsAnyOfMember(bool) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithAnyOfMemberIndex(int) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithIsTupleMember(bool) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithTupleMemberIndex(int) SchemaBuilder {
	return b
}

// IsTupleAdditionalItems indicates if a schema is located in the additionalItems (or items) section of a tuple schema.
func (b SchemaBuilder) WithIsTupleAdditionalItems(bool) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithChildren([]AnalyzedSchema) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithKind(kind analyzers.SchemaKind) SchemaBuilder {
	b.s.kind = kind

	return b
}

func (b SchemaBuilder) WithScalarKind(scalarKind analyzers.ScalarKind) SchemaBuilder {
	b.s.scalarKind = scalarKind

	return b
}

// IsEnum is a schema that boils down (after reduction) to a const or enum.
func (b SchemaBuilder) WithEnum(enum []dynamic.JSON) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithExtensions(extensions Extensions) SchemaBuilder {
	b.s.extensions = extensions

	return b
}

func (b SchemaBuilder) WithOriginalName(original string) SchemaBuilder {
	b.s.originalName = original

	return b
}

func (b SchemaBuilder) WithFormatValidation(format string) SchemaBuilder {
	return b
}

func (b SchemaBuilder) WithPattern(string) SchemaBuilder {
	return b
}

// IsAlwaysInvalid indicate that this schema is never valid.
func (b SchemaBuilder) WithIsAlwaysInvalid(bool) SchemaBuilder {
	return b
}
