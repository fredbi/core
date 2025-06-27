package jsonschema

import (
	"iter"

	"github.com/fredbi/core/json"
)

type SchemaType uint8

const (
	SchemaTypeNull SchemaType = iota
	SchemaTypeObject
	SchemaTypeArray
	SchemaTypeString
	SchemaTypeNumber
	SchemaTypeInteger
	SchemaTypeBool
)

type Schema struct {
	json.Document

	// decoded syntax
	source string
}

func Make(opts ...Option) Schema {
	// TODO: default store
	return Schema{} // TODO
}

func New(opts ...Option) *Schema {
	s := Make(opts...)

	return &s
}

func (s Schema) IsDefined() bool {
	return false
}

func (s Schema) HasRef() bool {
	return false
}

func (s Schema) Ref() Ref {
	return Ref{}
}

func (s Schema) HasMultipleTypes() bool {
	return false
}

func (s Schema) Types() []SchemaType {
	return nil
}

func (s Schema) HasProperties() bool {
	return false
}

func (s Schema) Properties() iter.Seq2[string, Schema] {
	return nil
}

func (s Schema) HasDependentSchemas() bool {
	return false
}

func (s Schema) DependentSchemas() iter.Seq2[string, Schema] {
	return nil
}

func (s Schema) HasRequiredProperties() bool {
	return false
}

func (s Schema) RequiredProperties() iter.Seq[string] {
	return nil
}

func (s Schema) HasDependentRequiredProperties() bool {
	return false
}

func (s Schema) DependentRequiredProperties() iter.Seq2[string, string] {
	return nil
}

func (s Schema) HasArrayItems() bool {
	return false
}

func (s Schema) ArrayItems() Schema {
	return Schema{}
}

func (s Schema) HasTupleItems() bool {
	return false
}

func (s Schema) TupleItems() iter.Seq2[int, Schema] {
	return nil
}

func (s Schema) HasAdditionalProperties() bool {
	return false
}

func (s Schema) AdditionalProperties() Schema {
	return Schema{}
}

func (s Schema) HasTupleAdditionalItems() bool {
	return false
}

func (s Schema) TupleAdditionalItems() Schema {
	return Schema{}
}

func (s Schema) HasPatternProperties() bool {
	return false
}

func (s Schema) PatternProperties() iter.Seq2[string, Schema] {
	return nil
}

func (s Schema) HasAllOf() bool {
	return false
}

func (s Schema) AllOf() iter.Seq2[int, Schema] {
	return nil
}

func (s Schema) HasAnyOf() bool {
	return false
}

func (s Schema) AnyOf() iter.Seq2[int, Schema] {
	return nil
}

func (s Schema) HasOneOf() bool {
	return false
}

func (s Schema) OneOf() iter.Seq2[int, Schema] {
	return nil
}

func (s Schema) Not() Schema {
	return Schema{}
}

func (s Schema) HasIfThenElse() bool {
	return false
}

func (s Schema) IfThenElse() (ifSchema Schema, thenSchema Schema, elseSchema Schema) {
	return
}

func (s Schema) HasNumberValidations() bool {
	return false
}

func (s Schema) NumberValidations() iter.Seq[NumberValidation] {
	// minimum, maximum, multipleOf
	// format (int64, ...)
	return nil
}

func (s Schema) HasStringValidations() bool {
	return false
}

func (s Schema) StringValidations() iter.Seq[StringValidation] {
	// minLength, maxLength, pattern
	// format, contentMediaType, contentEncoding, contentSchema
	return nil
}

func (s Schema) HasObjectValidations() bool {
	return false
}

func (s Schema) ObjectValidations() iter.Seq[ObjectValidation] {
	// minProperties, maxProperties, propertyNames
	return nil
}

func (s Schema) HasArrayValidations() bool {
	return false
}

func (s Schema) ArrayValidations() iter.Seq[ArrayValidation] {
	// minItems, maxItems, uniqueItems
	return nil
}

func (s Schema) HasEnumValidations() bool {
	return false
}

func (s Schema) EnumValidations() iter.Seq[EnumValidation] {
	// enum or const
	return nil
}

func (s Schema) EnumValues() iter.Seq[json.Document] {
	return nil
}

func (s Schema) HasDefaultValue() bool {
	return false
}

func (s Schema) DefaultValue() json.Document {
	return json.EmptyDocument
}

func (s Schema) ConstValue() json.Document {
	return json.EmptyDocument
}

func (s Schema) HasExtensions() bool {
	return false
}

func (s Schema) Extensions() iter.Seq2[string, json.Document] {
	// x-* extra keys
	return nil
}

func (s Schema) HasExtraKeys() bool {
	return false
}

func (s Schema) ExtraKeys() iter.Seq2[string, json.Document] {
	// other extra keys not recognized as extensions
	return nil
}

func (s Schema) HasMetadata() bool {
	return false
}

func (s Schema) Metadata() Metadata {
	// title, description, examples, $deprecated, $id, readOnly, writeOnly, $comment
	return Metadata{}
}

func (s Schema) VersionRequirements() VersionRequirements {
	return VersionRequirements{}
}
