package providers

import (
	"errors"
	"fmt"
	"path"
	"strconv"

	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

// UniqueSchema yields the key that should be considered unique for schema names.
//
// For schemas, what we want to be unique across a namespace (i.e. after generation, a go package), is
// the go type identifier.
func (p *NameProvider) UniqueSchema(name string) structural.Ident {
	return structural.Ident(p.mangler.ToGoName(name))
}

// NameSchema knows how to determine the go type name for a schema, when called back by the analyzer.
//
// Original schema names from JSON are mangled into go names.
//
// Anonymous sub-schemas may be named according to the context in which they are found.
//
// The extension "x-go-name" allows users to define directly the type name.
//
// TODO(fred): consider naming strategy based on title if present
func (p *NameProvider) NameSchema(
	name string,
	analyzed structural.AnalyzedSchema,
) (goName string, err error) {
	did, logAudit := p.prepareAuditSchema(&analyzed)
	defer logAudit()

	if p.marker != nil {
		// document the schema with the original schema name
		defer func() {
			if goName == name || analyzed.IsAnonymous() {
				return
			}

			mark := make(structural.Extensions, 1)
			mark.Add("x-go-original-name", analyzed.Name())
			p.marker.MarkSchema(analyzed, mark)
		}()
	}

	// apply explicit user directive
	const directive = "x-go-name"
	if ext, isUserDefined := analyzed.GetExtension(directive); isUserDefined {
		goName = ext.(string)

		if analyzed.IsAnonymous() {
			did(structural.AuditActionNameAnonymous, fmt.Sprintf(
				"applied directive %s to force schema name to %q", directive, goName,
			))
		} else {
			did(structural.AuditActionRenameSchema, fmt.Sprintf(
				"applied directive %s to rename schema name %q to %q", directive, name, goName,
			))
		}

		return goName, nil
	}

	// named schema
	if name != "" {
		goName = p.mangler.ToGoName(name)

		did(structural.AuditActionRenameSchema, fmt.Sprintf(
			"applied mangler ToGoName to transform %q into %q", name, goName,
		))

		return goName, nil
	}

	// special case of a named schema being refactored by the analyzer (here the name argument is "")
	if !analyzed.IsAnonymous() {
		if analyzed.IsRefactored() {
			// TODO: depending on the refactoring action find a better name
			// example: "{name}Without{prop1}{prop2}
			did(structural.AuditActionNameInfo,
				"inspected anonymous refactored schema but did nothing special",
			)
		}

		// other cases (???)
		return p.mangler.ToGoName(analyzed.Name()), nil
	}

	// moving forward, we proceed with anonymous schemas only

	if analyzed.IsRoot() {
		// we don't have any parent, so switch to alternate method to define a name
		goName, err = p.findNameForAnonymousRoot(analyzed, did)

		return goName, err
	}

	// descope things for which we'll use native types or strfmt types
	if p.dontGenerateTypeFor(analyzed) {
		// we have a parent, and some types may stay anonymous (e.g. primitive type, ...)
		did(structural.AuditActionNameInfo, "don't generate a type for this schema")

		return "", nil
	}

	// moving forward, schemas are anonymous and have a parent schema
	goName, err = p.nameAnonymousChild(analyzed, did)

	return goName, err
}

// DeconflictSchema deconflicts two schema names that get the same go type identifier.
//
// Example:
//
//	definitions:
//	  cloud_nodes: {}
//	  cloudNodes: {}
//
// Would both get the same CloudNodes go type name.
//
// Name conflicts may also occur when the bundling strategy attempts to flatten namespaces.
//
// At present, we just apply a simplistic strategy that adds a numerical index.
// This would result in the above example in:
//
//   - CloudNodes
//   - CloudNodes2
func (p *NameProvider) DeconflictSchema(
	name string,
	namespace structural.Namespace,
) (goName string, err error) {
	meta, _ := namespace.Meta(p.UniqueSchema(name))
	analyzed := meta.Schema
	did, logAudit := p.prepareAuditSchema(analyzed)
	defer logAudit()

	// insert more elaborate strategies here

	// simple deconfliction strategy that always works
	for idx := range maxAttempts {
		goName = p.mangler.ToGoName(name + " " + strconv.Itoa(idx+indexingStart))
		if namespace.CheckNoConflict(p.UniqueSchema(goName)) {
			did(structural.AuditActionRenameSchema, fmt.Sprintf(
				"deconflicted schema name using degraded naming strategy, got %q then iterated index to %d: %q",
				name,
				idx,
				goName,
			))

			return goName, nil
		}
	}

	// unless we hit an internal error or bug, we never get there: panic
	assertMustDeconflictUsingIndex(name)

	// never get there: we want alias deconfliction to always complete
	return "", nil
}

func (p *NameProvider) nameAnonymousChild(
	analyzed structural.AnalyzedSchema,
	did auditFunc,
) (goName string, err error) {
	// determine a name for an anonymous schema, which is not a root
	parent := analyzed.Parent()

	// walk up dependencies until we find a named schema
	//
	// TODO: handle chains of anonymous stuff may generate bizarre stuff and conflicts !!!
	parentName, err := p.NameSchema(parent.Name(), parent)
	if err != nil || parentName == "" {
		return "", fmt.Errorf(
			"tried to infer a name from its parent, but was unable to determine a name for the parent of this schema: %v: %w",
			analyzed,
			err,
		)
	}

	switch {
	// anonymous schema declared as a property of a parent object (which is not a polymorph)
	//
	// NOTE: this includes allOf members of an object and sub-types.
	case parent.IsObject() && !parent.IsPolymorphic():
		return p.nameAnonymousSchemaInObject(parent, analyzed, parentName, did)

	// anonymous schema declared as an items schema of a parent array
	//
	// NOTE: this includes allOf members of an array
	case parent.IsArray() && !parent.IsPolymorphic():
		return p.nameAnonymousSchemaInArray(parent, analyzed, parentName, did)

	// anonymous schema declared as an items schema of a parent tuple
	//
	// NOTE: this includes allOf members of a tuple
	case parent.IsTuple() && !parent.IsPolymorphic():
		return p.nameAnonymousSchemaInTuple(parent, analyzed, parentName, did)

	// anonymous schema declared as an items schema of a parent which is "anyOf" or "oneOf"
	case parent.IsPolymorphic():
		return p.nameAnonymousSchemaInPolymorph(parent, analyzed, parentName, did)

	default:
		// TODO: code assertion - other cases are invalid JSON schema or unexpected outcome of a refactored schema
		return "", fmt.Errorf(
			"hit unsupported case of an anonymous, non-root schema: %v: %w",
			analyzed, ErrNotImplemented,
		)
	}
}

func (p *NameProvider) dontGenerateTypeFor(analyzed structural.AnalyzedSchema) bool {
	// cases when we don't need to define a name:
	// * scalar values are mapped as primitive types (or format types)
	// * unconstrained types (without type-specific validations) are mapped as "any"
	return analyzed.IsNull() || analyzed.IsScalar() || analyzed.IsAnyWithoutValidation() ||
		analyzed.IsAlwaysInvalid()
}

func (p *NameProvider) findNameForAnonymousRoot(
	analyzed structural.AnalyzedSchema,
	did func(structural.AuditAction, string),
) (goName string, err error) {
	if analyzed.HasSchemaID() {
		name := path.Base(analyzed.SchemaID())
		goName = p.mangler.ToGoName(name)

		did(structural.AuditActionRenameSchema, fmt.Sprintf(
			"case of an anonymous root schema. Since no named parent exists, infer a name after the schema $id: %q",
			goName,
		))

		return goName, nil
	}

	// alternate strategy for anonymous root schema without any $id, e.g. "Object", "Array"...
	switch {
	case analyzed.IsScalar():
		goName = p.mangler.ToGoName(analyzed.ScalarKind().String())
	case analyzed.IsObject():
		goName = p.mangler.ToGoName("object")
	case analyzed.IsArray():
		goName = p.mangler.ToGoName("array")
	case analyzed.IsTuple():
		goName = p.mangler.ToGoName("tuple")
	case analyzed.IsAnyWithoutValidation():
		goName = p.mangler.ToGoName("any")
	case analyzed.IsEnum():
		goName = p.mangler.ToGoName("enum")
	case analyzed.IsPolymorphic():
		// TODO: implement case for polymorphic schemas
		return "", fmt.Errorf(
			"not implemented anonymous polymorphic root schema: %w",
			ErrNotImplemented,
		)

	default:
		return "", fmt.Errorf("not implemented anonymous root schema: %w", ErrNotImplemented)
	}

	did(structural.AuditActionRenameSchema, fmt.Sprintf(
		"case of an anonymous root schema. Since no named parent exists, infer a name after the schema type: %q",
		goName,
	))

	return goName, nil
}

func (p *NameProvider) nameAnonymousSchemaInObject(
	parent, analyzed structural.AnalyzedSchema,
	parentName string,
	did auditFunc,
) (goName string, err error) {
	_ = parent

	switch {
	case analyzed.HasParentProperty():
		// case with properties
		//
		// parent:
		//   type: object
		//   properties:
		//     propertyName: {analyzed}   <- ParentProperty() = "propertyName"
		//
		// Yields: "ParentPropertyName"
		propertyName := analyzed.ParentProperty()
		goName = p.mangler.ToGoName(parentName + " " + propertyName)
		did(structural.AuditActionNameAnonymous, fmt.Sprintf(
			"derived name from parent and property, got %q and %q, then applied mangler ToGoName: %q",
			parentName,
			propertyName,
			goName,
		))

		return goName, nil

	case analyzed.IsAdditionalProperty():
		// case with properties and additional properties
		//
		// parent:
		//   type: object
		//   properties:
		//     propertyName: {}
		//   additionalProperties: {analyzed}
		//
		// Yields: "ParentAdditionalProperties"
		if parent.NumAllProperties() > 0 { // also counts implicit properties (i.e. presence of pattern properties, non-explicit required, ...
			goName = p.mangler.ToGoName(parentName + " additional properties")
			did(structural.AuditActionNameAnonymous, fmt.Sprintf(
				`derived name from parent, got %q, added "additional properties", then applied mangler ToGoName: %q`,
				parentName,
				goName,
			))

			return goName, nil
		}

		// case with only additional properties
		//
		// parent:
		//   type: object
		//   additionalProperties: {analyzed}
		//
		// Yields: "ParentProperties"
		//
		// Target type would be something like "type Parent map[string]ParentProperties"
		goName = p.mangler.ToGoName(parentName + " properties")
		did(structural.AuditActionNameAnonymous, fmt.Sprintf(
			`derived name from parent, got %q, since we only have additional properties, added "properties", then applied mangler ToGoName: %q`,
			parentName,
			goName,
		))

		return goName, nil

		// NOTE:  we don't have this at this stage:
		// parent:
		//   type: object
		//   additionalProperties: { true (analyzed) }
		//
		// Because the analyzed schema is selected to remain anonymous (mapped to "any")

	case analyzed.IsPatternProperty():
		// case with patternProperties
		//
		// parent:
		//   type: object
		//   patternProperties:
		//    "regexp1": { analyzed }}
		//    "regexp2": { ... }}
		//
		// Yields: "ParentPatternProperties0"
		//
		// Possible alternatives to contemplate in the future:
		// - name based on analyzed schema type: e.g. ObjectProperties, NumberProperties ...
		// - AI-powered regexp summarizer ?
		//
		// NOTE: "propertyNames" do not add structure semantics, only validation
		index := analyzed.PatternPropertyIndex()
		goName = p.mangler.ToGoName(parentName + " pattern properties" + strconv.Itoa(index))
		did(structural.AuditActionNameAnonymous, fmt.Sprintf(
			`derived name from parent, got %q, added "pattern properties" and the index %d, then applied mangler ToGoName: %q`,
			parentName,
			index,
			goName,
		))

		return goName, nil

	case analyzed.IsAllOfMember():
		// case with parent object defined with allOf
		//
		// parent:
		//   type: object  <- possibly implicit
		//   allOf:
		//     - {...}
		//     - { analyzed }
		//     - {...}
		//
		// Yields: "ParentsInteger"ParentAllOf1"
		//
		// TODO: assertion - after analysis, we don't have stuff like
		// parent:
		//   type: object
		//   allOf: [ { a }, { b } ]
		//   oneOf: [ { c }, { d } ]
		//
		// This would be rewritten by the analyzer as:
		// parent:
		//   type: object
		//   allOf:
		//     - { a }
		//     - { b }
		//     - oneOf: [ { c }, { d } ]
		//
		// TODO: assertion - after analysis, we don't have allOf with 1 member only (lifted)
		// -> may be the case for named object
		//
		// TODO: assertion - after analysis, anonymous allOf members are lifted whenever possible. TODO
		// find the cases when this is not possible to lift.
		// Perhaps edges cases with "additionalPropertie: false" and "unevaluatedProperties"
		//
		// Meaning that we only one of "allOf", "oneOf" or "anyOf" to consider at any schema level.
		if analyzed.IsSubType() {
			// edge case with an anonymous declaration of a subttype.
			// TODO: we should be able to find better than "allOf0"
			index := analyzed.AllOfMemberIndex()
			goName = p.mangler.ToGoName(parentName + " allOf" + strconv.Itoa(index))
			did(structural.AuditActionNameAnonymous, fmt.Sprintf(
				`derived name from parent, got %q, added "allOf" and the index %d, then applied mangler ToGoName: %q`,
				parentName,
				index,
				goName,
			))

			return goName, nil
		}
		return p.mangler.ToGoName(
			parentName + " allOf" + strconv.Itoa(analyzed.AllOfMemberIndex()),
		), nil

	default:
		// TODO: analyzer - at the moment, we assume that it is possible to lift all anonymous members of an object, but this
		// is not obvious. For instance consider this:
		//
		// parent:
		//   type: object
		//   allOf:
		//     - anyOf: [{}, {}]
		//     - $ref: #/X
		//
		// I am not really sure we can do this, because we forbid this construct:
		// parent:
		//   type: object
		//   anyOf: [{}, {}]
		//   allOf:
		//     - $ref: #/X
		assertAnonymousInParentObject(analyzed)

		// TODO: code assertion, panic not error
		return "", fmt.Errorf(
			`hit an unsupported case of the analyzed schema. Some assumptions about the contract guaranteed by %[1]T where not met: %[1]v: %w`,
			analyzed,
			ErrInternal,
		)
	}
}

func (p *NameProvider) nameAnonymousSchemaInArray(
	parent, analyzed structural.AnalyzedSchema,
	parentName string,
	did auditFunc,
) (goName string, err error) {
	_ = parent

	if p.annotator != nil && !analyzed.Metadata().HasTitle() {
		// injects a title for the array items type if missing
		defer func() {
			if err != nil {
				title := fmt.Sprintf(
					`%s is an element of [%s]`,
					goName,
					p.mangler.ToGoName(parentName),
				)

				metaBuilder := structural.NewMetadataBuilder().From(analyzed).WithTitle(title)
				p.annotator.AnnotateSchema(analyzed, metaBuilder.Metadata())
			}
		}()
	}

	switch {
	// Items in an array
	//
	// parent:
	//   type: array
	//   items: {analyzed}
	//
	// Yields: "ParentItems"
	case analyzed.IsItems(): // implicit items don't get there (remain anonymous)
		goName = p.mangler.ToGoName(parentName + " items")
		did(structural.AuditActionNameAnonymous, fmt.Sprintf(
			`derived array items name from parent, got %q, added "items", then applied mangler ToGoName: %q`,
			parentName,
			goName,
		))

		return goName, nil

	case analyzed.IsAllOfMember():
		// AllOf in an array: we should not get there.
		//
		// parent:
		//   type: array
		//   allOf:
		//     - type: array
		//       items: {...}
		//     - type: array   <- { analyzed }
		//       items: {...}
		//
		// Yields: "ParentItems"
		//
		// Assertion - the analyzer transforms constructs such as
		//
		// parent:
		//   type: array
		//   items: { a }
		//   allOf:
		//     - { b }
		//     - { c }
		//
		// or:
		// parent:
		//   type: array
		//   allOf:
		//     - type: array
		//       items: { a }
		//     - { b }
		//     - { c }
		//
		// Into:
		//
		// parent:
		//   type: array
		//   items:
		//     allOf:
		//       - { a }
		//       - { b }
		//       - { c }
		//   { merged array validations }
		assertAllOfInParentArray(analyzed)

		return "", errors.ErrUnsupported

	default:
		assertAnonymousInParentArray(analyzed)

		return "", errors.ErrUnsupported
	}
}

func (p *NameProvider) nameAnonymousSchemaInTuple(
	parent, analyzed structural.AnalyzedSchema,
	parentName string,
	did auditFunc,
) (goName string, err error) {
	_ = parent

	switch {
	case analyzed.IsTupleMember():
		// parent:
		//   type: array
		//   items: (prefixItems:)
		//     - {analyzed}
		//
		// Yields: "ParentItems0"
		goName = p.mangler.ToGoName(
			parentName + " items" + strconv.Itoa(analyzed.TupleMemberIndex()),
		)
		did(structural.AuditActionNameAnonymous, fmt.Sprintf(
			`derived tuple name from parent, got %q, added "items", then applied mangler ToGoName: %q`,
			parentName,
			goName,
		))

		return goName, nil

	case analyzed.IsTupleAdditionalItems():
		// parent:
		//   type: array
		//   items: (prefixItems:)
		//     - {...}
		//     - {...}
		//   additionalItems: {analyzed} (items:)
		//
		// Yields: "ParentAdditionalItems"
		goName = p.mangler.ToGoName(parentName + " additional items")
		did(structural.AuditActionNameAnonymous, fmt.Sprintf(
			`derived tuple additional items name from parent, got %q, added "additional items", then applied mangler ToGoName: %q`,
			parentName,
			goName,
		))

		return goName, nil

	// NOTE: edge case where allOf is rewritten like for arrays (after evaluated not be always false)
	// parent:
	//   type: array
	//   items: (prefixItems:)
	//     - {...}
	//     - {...}
	//   allOf:
	//     - items:
	//         - {...}
	//         - {...}
	//   additionalItems: {...} (items:)
	//
	// Is either invalid (always evaluate to false) or may be rewritten

	default:
		assertAnonymousInParentTuple(analyzed)

		return "", errors.ErrUnsupported
	}
}

func (p *NameProvider) nameAnonymousSchemaInPolymorph(
	parent, analyzed structural.AnalyzedSchema,
	parentName string,
	did auditFunc,
) (goName string, err error) {
	_ = parent

	switch {
	case analyzed.IsOneOfMember():
		// case with oneOf, anyOf
		//
		// parent:
		//   type: object
		//   oneOf:
		//     - {analyzed}
		//
		// Yields: "ParentAllOf0"
		index := analyzed.OneOfMemberIndex()
		goName = p.mangler.ToGoName(
			parentName + " one of" + strconv.Itoa(index),
		)
		did(structural.AuditActionNameAnonymous, fmt.Sprintf(
			`derived oneOf member name from parent, got %q, added "one of" and index %d , then applied mangler ToGoName: %q`,
			parentName,
			index,
			goName,
		))

		return goName, nil

	case analyzed.IsAnyOfMember():
		index := analyzed.AnyOfMemberIndex()
		goName = p.mangler.ToGoName(
			parentName + " any of" + strconv.Itoa(index),
		)
		did(structural.AuditActionNameAnonymous, fmt.Sprintf(
			`derived anyOf member name from parent, got %q, added "any of" and index %d , then applied mangler ToGoName: %q`,
			parentName,
			index,
			goName,
		))

		return goName, nil

		/*
			/* TODO: this seems invalid
			case analyzed.IsAllOfMember() && analyzed.IsSubType():
				// case of an anonymous subtype
				//
				// parent:
				//   type: object
				//   ...
				//   anyOf:
				//     - { ... }
				//     - type: object   <- { analyzed: subtype }
				//       properties: { ... }
				//       allOf:
				//         - $ref: #/definition/BaseType
				//         - { ... }
				// -> anonymous allOf should be lifted
				return p.mangler.ToGoName(parentName + " all of" + strconv.Itoa(analyzed.AllOfMemberIndex())), nil
		*/
	default:
		assertAnonymousInParentPolymorphic(analyzed)

		return "", errors.ErrUnsupported
	}
}
