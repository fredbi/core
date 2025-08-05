package jsonschema

import (
	"iter"
	"slices"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/nodes/light"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/swag/typeutils"
)

// Validation captures the JSON schema vocabulary considered as "validation" by JSON schema draft 2020.
//
// This includes the following JSON schema keywords:

// General validations:
//
//   - type
//   - const (>= draft 6)
//   - enum
//   - $data (if option enabled)
//
// String validations:
//
//   - maxLength
//   - minLength
//   - pattern
//   - format
//   - contentSchema (>= draft 2019)
//
// Number validations:
//
//   - exclusiveMaximum (>=draft6: bool to number)
//   - exclusiveMinimum (>=draft6: bool to number)
//   - format (format may apply to string or numbers)
//   - maximum
//   - minimum
//   - multipleOf
//
// Object validations:
//
//   - maxProperties
//   - minProperties
//   - required (< draft 6: array must not be empty)
//   - dependentRequired (>= draft 2019 ; pre-draft 2019: dependencies as array)
//
// Array validations:
//
//   - maxContains (>= draft 2019)
//   - maxItems
//   - minContains (>= draft 2019)
//   - minItems
//   - uniqueItems
//
// Note: "format" is formally considered an "annotation" by the JSON schema specification.
type Validation struct {
	defined bool
	types   []SchemaType

	numberV []NumberValidation
	stringV []StringValidation
	objectV []ObjectValidation
	arrayV  []ArrayValidation
	enumV   []EnumValidation
}

var (
	typeKey = values.MakeInternedKey("type")
	dataKey = values.MakeInternedKey("$data")

	enumKey  = values.MakeInternedKey("enum")
	constKey = values.MakeInternedKey("const")

	enumValidationKeys = map[values.InternedKey]struct{}{
		enumKey:  {},
		constKey: {},
	}

	contentSchemaKey = values.MakeInternedKey("contentSchema")
	maxLengthKey     = values.MakeInternedKey("maxLength")
	minLengthKey     = values.MakeInternedKey("minLength")
	patternKey       = values.MakeInternedKey("pattern")
	formatKey        = values.MakeInternedKey("format")

	stringValidationKeys = map[values.InternedKey]struct{}{
		contentSchemaKey: {},
		formatKey:        {},
		maxLengthKey:     {},
		minLengthKey:     {},
		patternKey:       {},
	}

	exclusiveMaximumKey = values.MakeInternedKey("exclusiveMaximum")
	exclusiveMinimumKey = values.MakeInternedKey("exclusiveMinimum")
	maximumKey          = values.MakeInternedKey("maximum")
	minimumKey          = values.MakeInternedKey("minimum")
	multipleOfKey       = values.MakeInternedKey("multipleOf")

	numberValidationKeys = map[values.InternedKey]struct{}{
		exclusiveMaximumKey: {},
		exclusiveMinimumKey: {},
		formatKey:           {},
		maximumKey:          {},
		minimumKey:          {},
		multipleOfKey:       {},
	}

	maxPropertiesKey     = values.MakeInternedKey("maxProperties")
	minPropertiesKey     = values.MakeInternedKey("minProperties")
	requiredKey          = values.MakeInternedKey("required")
	dependenciesdKey     = values.MakeInternedKey("dependencies")
	dependentRequiredKey = values.MakeInternedKey("dependentRequired")

	objectValidationKeys = map[values.InternedKey]struct{}{
		maxPropertiesKey:     {},
		minPropertiesKey:     {},
		requiredKey:          {},
		dependenciesdKey:     {},
		dependentRequiredKey: {},
	}

	maxContainsKey = values.MakeInternedKey("maxContains")
	maxItemsKey    = values.MakeInternedKey("maxItems")
	minContainsKey = values.MakeInternedKey("minContains")
	minItemsKey    = values.MakeInternedKey("minItems")
	uniqueItemsKey = values.MakeInternedKey("uniqueItems")

	arrayValidationKeys = map[values.InternedKey]struct{}{
		maxContainsKey: {},
		maxItemsKey:    {},
		minContainsKey: {},
		minItemsKey:    {},
		uniqueItemsKey: {},
	}

	validationKeys = typeutils.MergeMaps(nil,
		map[values.InternedKey]struct{}{
			typeKey: {},
			dataKey: {},
		},
		enumValidationKeys,
		stringValidationKeys,
		numberValidationKeys,
		objectValidationKeys,
		arrayValidationKeys,
	)

	validationConstraints = map[values.InternedKey]VersionRequirements{
		constKey:             {MinVersion: VersionDraft6},
		contentSchemaKey:     {MinVersion: VersionDraft2019},
		dependenciesKey:      {MaxVersion: VersionDraft7}, // TODO: check if StrictMaxVersion
		dependentRequiredKey: {MinVersion: VersionDraft2019},
		maxContainsKey:       {MinVersion: VersionDraft2019},
		minContainsKey:       {MinVersion: VersionDraft2019},
	}
)

func (v *Validation) decode(ctx *light.ParentContext, key values.InternedKey, vr *VersionRequirements) error {
	return nil // TODO
}

type NumberValidation struct{}
type StringValidation struct{}
type ObjectValidation struct{}
type ArrayValidation struct{}
type EnumValidation struct {
	defined bool
}

func (e EnumValidation) IsDefined() bool {
	return e.defined
}

func (v Validation) IsDefined() bool {
	return v.defined
}

func (v Validation) HasType() bool {
	return len(v.types) > 0
}

func (v Validation) HasMultipleTypes() bool {
	return len(v.types) > 1
}

func (v Validation) Types() []SchemaType {
	return v.types
}

func (v Validation) HasNumberValidations() bool {
	return len(v.numberV) > 0
}

func (v Validation) NumberValidations() iter.Seq[NumberValidation] {
	return slices.Values(v.numberV)
}

func (v Validation) HasStringValidations() bool {
	return len(v.stringV) > 0
}

func (v Validation) StringValidations() iter.Seq[StringValidation] {
	return slices.Values(v.stringV)
}

func (v Validation) HasObjectValidations() bool {
	return len(v.objectV) > 0
}

func (v Validation) ObjectValidations() iter.Seq[ObjectValidation] {
	return slices.Values(v.objectV)
}

func (v Validation) HasArrayValidations() bool {
	return len(v.arrayV) > 0
}

func (v Validation) ArrayValidations() iter.Seq[ArrayValidation] {
	return slices.Values(v.arrayV)
}

func (v Validation) HasEnumValidations() bool {
	return len(v.enumV) > 0
}

func (v Validation) EnumValidations() iter.Seq[EnumValidation] {
	return slices.Values(v.enumV)
}

func (v EnumValidation) EnumValues() iter.Seq[json.Document] {
	return nil
}

func (v EnumValidation) ConstValue() json.Document {
	return json.EmptyDocument
}
