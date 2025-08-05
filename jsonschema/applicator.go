package jsonschema

import (
	"iter"

	"github.com/fredbi/core/json/nodes/light"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/stores/values"
)

// Applicator captures the JSON schema vocabulary considered as "applicators" by JSON schema draft 2020.
//
// This includes the following JSON schema keywords:
//
//   - additionalProperties
//   - allOf
//   - anyOf
//   - contains (>= draft 6)
//   - dependentSchemas (>= draft 2019 ; pre-draft 2019: dependencies)
//   - if, then, else: (>= draft 7)
//   - items (for tuples: >= draft 2020 ; pre-draft 2020: additionalItems)
//   - not
//   - oneOf
//   - patternProperties
//   - prefixItems (>= draft 2020 ; pre-draft 2020: items as array for tuples)
//   - properties
//   - propertyNames (>= draft 6)
//   - unevaluatedItems (>= draft 2019)
//   - unevaluatedProperties (>= draft 2019)
//
// For OpenAPI schemas:
//   - discriminator (>= OpenAPI v2)
type Applicator struct {
	s                    stores.Store
	additionalProperties *light.Node
	allOf                []*light.Node
	anyOf                []*light.Node
	contains             *light.Node
	//...
	properties []*light.Node
	defined    bool
}

var (
	additionalPropertiesKey  = values.MakeInternedKey("additionalProperties")
	allOfKey                 = values.MakeInternedKey("allOf")
	anyOfKey                 = values.MakeInternedKey("anyOf")
	containsKey              = values.MakeInternedKey("contains")
	dependentSchemasKey      = values.MakeInternedKey("dependentSchemas")
	dependenciesKey          = values.MakeInternedKey("dependencies")
	discriminatorKey         = values.MakeInternedKey("discriminator")
	ifKey                    = values.MakeInternedKey("if")
	thenKey                  = values.MakeInternedKey("then")
	elseKey                  = values.MakeInternedKey("else")
	itemsKey                 = values.MakeInternedKey("items")
	notKey                   = values.MakeInternedKey("not")
	oneOfKey                 = values.MakeInternedKey("oneOf")
	patternPropertiesKey     = values.MakeInternedKey("patternProperties")
	prefixItemsKey           = values.MakeInternedKey("prefixItems")
	additionalItemsKey       = values.MakeInternedKey("additionalItems")
	propertiesKey            = values.MakeInternedKey("properties")
	propertyNamesKey         = values.MakeInternedKey("propertyNames")
	unEvaluatedItemsKey      = values.MakeInternedKey("unevaluatedItems")
	unEvaluatedPropertiesKey = values.MakeInternedKey("unevaluatedProperties")

	applicatorKeys = map[values.InternedKey]struct{}{
		additionalPropertiesKey:  {},
		allOfKey:                 {},
		anyOfKey:                 {},
		containsKey:              {},
		dependentSchemasKey:      {},
		dependenciesKey:          {},
		discriminatorKey:         {},
		ifKey:                    {},
		thenKey:                  {},
		elseKey:                  {},
		itemsKey:                 {},
		notKey:                   {},
		oneOfKey:                 {},
		patternPropertiesKey:     {},
		prefixItemsKey:           {},
		additionalItemsKey:       {},
		propertiesKey:            {},
		propertyNamesKey:         {},
		unEvaluatedItemsKey:      {},
		unEvaluatedPropertiesKey: {},
	}

	applicatorConstraints = map[values.InternedKey]VersionRequirements{
		discriminatorKey:         {MinVersion: VersionOpenAPIv2},
		containsKey:              {MinVersion: VersionDraft6},
		propertyNamesKey:         {MinVersion: VersionDraft6},
		dependenciesKey:          {MaxVersion: VersionDraft7}, // TODO: check if StrictMaxVersion
		ifKey:                    {MinVersion: VersionDraft7},
		thenKey:                  {MinVersion: VersionDraft7},
		elseKey:                  {MinVersion: VersionDraft7},
		dependentSchemasKey:      {MinVersion: VersionDraft2019},
		unEvaluatedItemsKey:      {MinVersion: VersionDraft2019},
		unEvaluatedPropertiesKey: {MinVersion: VersionDraft2019},
		prefixItemsKey:           {MinVersion: VersionDraft2020},
	}
)

func (s Applicator) IsDefined() bool {
	return s.defined
}

func (s Applicator) HasAdditionalProperties() bool {
	return s.additionalProperties != nil
}

func (s Applicator) AdditionalProperties() Schema {
	return Schema{}
}

func (s Applicator) HasProperties() bool {
	return len(s.properties) > 0
}

func (a Applicator) Properties() iter.Seq2[string, Schema] {
	return nil
}

func (a Applicator) Property(_ string) (Schema, bool) {
	return Schema{}, false
}

func (a Applicator) HasDependentSchemas() bool {
	return false
}

func (a Applicator) DependentSchemas() iter.Seq2[string, Schema] {
	return nil
}

func (s Applicator) HasArrayItems() bool {
	return false
}

func (s Applicator) ArrayItems() Schema {
	return Schema{}
}

func (s Applicator) HasTuplePrefixItems() bool {
	return false
}

func (s Applicator) TuplePrefixItems() iter.Seq2[int, Schema] {
	return nil
}

func (s Applicator) HasTuplelItems() bool {
	return false
}

func (s Applicator) TupleItems() Schema {
	return Schema{}
}

func (s Applicator) HasPatternProperties() bool {
	return false
}

func (s Applicator) PatternProperties() iter.Seq2[string, Schema] {
	return nil
}

func (s Applicator) HasAllOf() bool {
	return false
}

func (s Applicator) AllOf() iter.Seq2[int, Schema] {
	return nil
}

func (s Applicator) HasAnyOf() bool {
	return false
}

func (s Applicator) AnyOf() iter.Seq2[int, Schema] {
	return nil
}

func (s Applicator) HasOneOf() bool {
	return false
}

func (s Applicator) OneOf() iter.Seq2[int, Schema] {
	return nil
}

func (s Applicator) HasNot() bool {
	return false
}

func (s Applicator) Not() Schema {
	return Schema{}
}

func (s Applicator) HasIfThenElse() bool {
	return false
}

func (s Applicator) IfThenElse() (ifSchema Schema, thenSchema Schema, elseSchema Schema) {
	return
}

func (s *Applicator) decode(ctx *light.ParentContext, key values.InternedKey, vr *VersionRequirements) error {
	octx, ok := ctx.X.(*schemaContext)
	if !ok {
		panic("bug")
	}

	switch key {
	case additionalPropertiesKey:
	case allOfKey:
	case anyOfKey:
	case containsKey:
	case dependentSchemasKey:
	case dependenciesKey:
	case discriminatorKey:
	case ifKey:
	case thenKey:
	case elseKey:
	case itemsKey:
	case notKey:
	case oneOfKey:
	case patternPropertiesKey:
	case prefixItemsKey:
	case additionalItemsKey:
	case propertiesKey:
	case propertyNamesKey:
	case unEvaluatedItemsKey:
	case unEvaluatedPropertiesKey:
	default:
		panic("bug")
	}

	return nil // TODO
}
