package providers

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/dynamic"
	"github.com/fredbi/core/json/lexers/token" // TODO: should alias token kinds somehow to avoid spreading this
	"github.com/fredbi/core/json/nodes"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
	"github.com/fredbi/core/swag/mangling"
)

// NameProvider provides go names for types and packages created from schemas.
//
// The [NameProvider] is not intended for concurrent use.
type NameProvider struct {
	options

	mangler         *mangling.NameMangler
	filesNamespaces map[string]structural.Namespace
}

// NewNameProvider builds a new [NameProvider] with possible options.
func NewNameProvider(opts ...Option) *NameProvider {
	p := NameProvider{
		options:         optionsWithDefaults(opts),
		filesNamespaces: make(map[string]structural.Namespace),
	}
	p.mangler = mangling.New(p.manglingOptions...)

	return &p
}

// EqualName compares names to ensure that no name conflict would occur.
func (p NameProvider) EqualName(a, b string) bool {
	return p.mangler.ToGoName(a) == p.mangler.ToGoName(b) || p.mangler.ToFileName(a) == p.mangler.ToFileName(b)
}

// EqualPath compares paths to ensure that no name conflict would occur.
func (p NameProvider) EqualPath(a, b string) bool {
	return p.mangler.ToGoPackagePath(a) == p.mangler.ToGoPackagePath(b)
}

// NameSchema knows how to determine the go type name for a schema, when called back by the analyzer.
//
// Original schema names from JSON are mangled into go names.
//
// Anonymous sub-schemas may be named according to the context in which they are found.
//
// The extension "x-go-name" allows users to define directly the type name.
func (p NameProvider) NameSchema(name string, analyzed structural.AnalyzedSchema) (string, error) {
	const directive = "x-go-name"

	if ext, isUserDefined := analyzed.GetExtension(directive); isUserDefined {
		goName := ext.(string)

		return goName, nil
	}

	if name != "" {
		return p.mangler.ToGoName(name), nil
	}

	if !analyzed.IsAnonymous() {
		return p.mangler.ToGoName(analyzed.Name), nil
	}

	if analyzed.IsRoot() {
		return p.findNameForAnonymousRoot(name, analyzed)
	}

	if p.dontGenerateTypeFor(analyzed) {
		return "", nil
	}

	// determine a name for an anonymous schema, which is not a root
	parent := analyzed.Parent()

	// walk up dependencies until we find a named schema
	parentName, err := p.NameSchema(parent.Name, parent)
	if err != nil {
		return "", fmt.Errorf("unable to determine a name for this schema: %v", analyzed)
	}

	if parentName == "" {
		return "", fmt.Errorf("unable to determine a name for this schema: %v", analyzed)
	}

	switch {
	// anonymous schema declared as a property of a parent object (which is not a polymorph)
	case parent.IsObject():
		// case with properties
		//
		// parent:
		//   type: object
		//   properties:
		//     propertyName: {analyzed}
		//
		// Yields: "ParentPropertyName"
		if analyzed.HasParentProperty() {
			propertyName := analyzed.ParentProperty()
			return p.mangler.ToGoName(parentName + " " + propertyName), nil
		}

		// case with additional properties
		//
		// parent:
		//   type: object
		//   additionalProperties: {analyzed}
		//
		// Yields: "ParentPropertyName"
		if analyzed.IsAdditionalProperty() {
			return p.mangler.ToGoName(parentName + " additional properties"), nil
		}

		assertAnonymousInParentObject(analyzed)
		return "", errors.ErrUnsupported

	// anonymous schema declared as an items schema of a parent array or tuple
	case parent.IsArray():
		//
		// parent:
		//   type: array
		//   items: {analyzed}
		//
		// Yields: "ParentItems"
		return p.mangler.ToGoName(parentName + " items"), nil

	case parent.IsTuple():
		//
		// parent:
		//   type: array
		//   items:
		//     - {analyzed}
		//
		// Yields: "ParentItems0"
		if seq := analyzed.IsTupleMember(); seq > 0 {
			return p.mangler.ToGoName(parentName + " items" + strconv.Itoa(seq)), nil // TODO: configurable suffix
		}
		if analyzed.IsTupleAdditionalItems() {
			return p.mangler.ToGoName(parentName + " additional items"), nil
		}

		assertAnonymousInParentTuple(analyzed)
		return "", errors.ErrUnsupported

	case parent.IsPolymorphic():
		// case with allOf, oneOf, anyOf
		//
		// parent:
		//   type: object
		//   allOf:
		//     - {analyzed}
		//
		// Yields: "ParentAllOf0"
		if seq := analyzed.IsAllOfMember(); seq > 0 {
			return p.mangler.ToGoName(parentName + " all of" + strconv.Itoa(seq)), nil
		}
		if seq := analyzed.IsOneOfMember(); seq > 0 {
			return p.mangler.ToGoName(parentName + " one of" + strconv.Itoa(seq)), nil
		}
		if seq := analyzed.IsAnyOfMember(); seq > 0 {
			return p.mangler.ToGoName(parentName + " any of" + strconv.Itoa(seq)), nil
		}

		assertAnonymousInParentPolymorphic(analyzed)
		return "", errors.ErrUnsupported

	default:
		// other cases are invalid JSON schema
		return "", errors.ErrUnsupported
	}
}

// NameEnumValue provides a legit go name for a constant or variable corresponding to a value in an enum.
//
// TODO: the user may opt-in to make (some of) these unexported.
func (p NameProvider) NameEnumValue(index int, enumValue json.Document, analyzed structural.AnalyzedSchema) (string, error) {
	// case with enum:
	// analyzed:
	//	 enum:
	//	 - x
	// 	 - y
	//   x-go-enums:
	//   - x-axis    -> XAxis
	//   - y-axis    -> YAxis
	//
	//	 enum:
	//	 - 1          -> One
	// 	 - 2.5        -> TwoPointFive
	//
	//	 enum:
	//	 - {1,2}      -> AnalyzedEnum0
	// 	 - [x,y]      -> AnalyzedEnum1
	//
	//	 enum:
	//	 - {1,2}      -> First
	// 	 - [x,y]      -> AnalyzedEnum1
	//   x-go-enums:
	//   - First
	//
	const directive = "x-go-enums"

	if ext, isUserDefined := analyzed.GetExtension(directive); isUserDefined {
		names := ext.([]string)

		if index < len(names) {
			return p.mangler.ToGoName(names[index]), nil
		}
	}

	switch enumValue.Kind() {
	case nodes.KindScalar:
		v, _ := enumValue.Value()
		switch v.Kind() {
		case token.String, token.Boolean, token.Null:
			return p.mangler.ToGoName(v.String()), nil // may be deconflicted later
		case token.Number:
			return p.mangler.ToGoName(p.mangler.SpellNumber(enumValue.String())), nil
		}
	default:
		// enum value is a complex schema, not a scalar
		// determine a name for an anonymous schema, which is not a root
		parent := analyzed.Parent()

		// walk up dependencies until we find a named schema
		parentName, err := p.NameSchema(parent.Name, parent)
		if err != nil {
			return "", fmt.Errorf("unable to determine a name for this schema: %v", analyzed)
		}

		if parentName == "" {
			return "", fmt.Errorf("unable to determine a name for this schema: %v", analyzed)
		}

		return p.mangler.ToGoName(parentName + " enum" + strconv.Itoa(index)), nil
	}

	return "", nil
}

// NamePackage knows how to determine the relative go package path for a schema, when called back by the analyzer.
//
// Example: generated/models/go-folder
//
// It rewrites names to get legit, idiomatic go package names:
//
// * x_test gets rewritten
// * x/v2 gets rewritten
// * Abc gets rewritten to abc
// * computeService gets rewritten as compute-service
// * compute_service gets rewritten as compute-service
func (p NameProvider) NamePackage(path string, analyzed structural.AnalyzedSchema) (string, error) {
	const directive = "x-go-package"

	if ext, isUserDefined := analyzed.GetExtension(directive); isUserDefined {
		goPkg := ext.(string)

		return goPkg, nil
	}

	return p.mangler.ToGoPackagePath(path), nil
}

// PackageShortName provides the package name to be used in the "package" statement.
//
// Examples:
// * generated/models/go-folder -> "folder"
// * generated/models/go-folder/v2 -> "folder"
func (p NameProvider) PackageShortName(path string, analyzed structural.AnalyzedSchema) string {
	return p.mangler.ToGoPackageName(path)
}

// PackageFullName returns the fully qualified package name, to be used in imports.
//
// Example: generated/models/go-folder -> "github.com/fredbi/core/genmodels/generated/models/go-folder"
func (p NameProvider) PackageFullName(path string, analyzed structural.AnalyzedSchema) string {
	return "" // TODO
}

// Mark the analyzed schema with trailing information.
func (p NameProvider) Mark(analyzed structural.AnalyzedSchema) structural.Extensions {
	// TODO: this is just an example
	mark := make(structural.Extensions, 1)

	mark.Add("x-go-original-name", analyzed.Name)
	mark.Add("x-go-original-path", analyzed.Path)

	return mark
}

// FileName produces a source file name to hold model code.
//
// It is possible to override a generated file name using "x-go-file-name".
//
// # FileName produces legit, idiomatic file names
//
// xyz_unix gets rewritten
// xyz_test gets rewritten
// Abc XYZ becomes abc-xyz
func (p NameProvider) FileName(name string, analyzed structural.AnalyzedSchema) string {
	const directive = "x-go-file-name"

	if ext, isUserDefined := analyzed.GetExtension(directive); isUserDefined {
		goFile := ext.(string)

		if p.isFileConflict(goFile, analyzed.Path) {
			return p.deconflictsFile(goFile, analyzed.Path)
		}

		p.registerFile(goFile, analyzed.Path)

		return goFile
	}

	goFile := p.mangler.ToFileName(name)
	if p.isFileConflict(goFile, analyzed.Path) {
		return p.deconflictsFile(goFile, analyzed.Path)
	}
	p.registerFile(goFile, analyzed.Path)

	return goFile
}

// FileNameForTest produces a source file name to hold test code.
func (p NameProvider) FileNameForTest(name string, analyzed structural.AnalyzedSchema) string {
	var suffix string
	if withoutTestSuffix, isTestFile := strings.CutSuffix(name, "_test"); isTestFile {
		name = withoutTestSuffix
		suffix = "_test"
	}

	goFile := p.mangler.ToFileName(name) + suffix
	if p.isFileConflict(goFile, analyzed.Path) {
		return p.deconflictsFile(goFile, analyzed.Path)
	}
	p.registerFile(goFile, analyzed.Path)

	return goFile
}

func (p NameProvider) dontGenerateTypeFor(analyzed structural.AnalyzedSchema) bool {
	// cases when we don't need to define a name:
	// * scalar values are mapped as primitive types
	// * unconstrained types (without type-specific validations) are mapped as "any"
	// * objects without properties and no constraints on additional properties are mapped as "any"
	return analyzed.IsScalar() || analyzed.IsAnyWithoutValidation() || (analyzed.IsObject() && analyzed.NumProperties() == 0)
}

func (p NameProvider) findNameForAnonymousRoot(name string, analyzed structural.AnalyzedSchema) (string, error) {
	if analyzed.DollarID != "" {
		name = path.Base(analyzed.DollarID)
		return p.mangler.ToGoName(name), nil
	}

	// alternate strategy for anonymous root schema without any $id, e.g. "Object", "Array"...
	switch { // TODO: would need more hints to help later deconfliction
	case analyzed.IsScalar():
		return p.mangler.ToGoName(analyzed.ScalarKind().String()), nil
	case analyzed.IsObject():
		return p.mangler.ToGoName("object"), nil
	case analyzed.IsArray():
		return p.mangler.ToGoName("array"), nil
	case analyzed.IsTuple():
		return p.mangler.ToGoName("tuple"), nil
	case analyzed.IsAnyWithoutValidation():
		return p.mangler.ToGoName("any"), nil
	case analyzed.IsEnum():
		return p.mangler.ToGoName("enum"), nil
	case analyzed.IsPolymorphic():
		// TODO
		return "", errors.New("not implemented anonymous polymorphic root schema")

	default:
		return "", errors.New("not implemented anonymous root schema")
	}
}

func (p NameProvider) registerFile(name, pth string) {
	namespace, ok := p.filesNamespaces[pth]
	if !ok {
		namespace = make(structural.Namespace)
	}

	namespace[name] = struct{}{}

	p.filesNamespaces[pth] = namespace

	return
}

// isFileConflict detects if the file name we are about to generate for this artifact
func (p NameProvider) isFileConflict(name, pth string) bool {
	namespace, ok := p.filesNamespaces[pth]
	if !ok {
		return false
	}
	_, alreadyExists := namespace[name]

	return alreadyExists
}

// deconflictsFile finds a deconflicted file name.
//
// The strategy to deconflict file names is simplistic:
//
// "object A" and "Object_a" identifiers would produce the same file target: object_a.
//
// The first would remain "object_a" and the next found on will be named "object_a_2".
func (p NameProvider) deconflictsFile(name, pth string) string {
	var suffix string

	if withoutTestSuffix, isTestFile := strings.CutSuffix(name, "_test"); isTestFile {
		name = withoutTestSuffix
		suffix = "_test"
	}

	for i := 1; ; i++ {
		attempt := name + "_" + strconv.Itoa(i) + suffix
		goFile := p.mangler.ToFileName(attempt)
		if p.isFileConflict(goFile, pth) {
			continue
		}

		p.registerFile(goFile, pth)

		return goFile
	}
}

// MapExtension maps extensions as directives into known go types.
//
// This is enforced by the analyzer, so later processing can rely on a safe typing for known extensions.
func (p NameProvider) MapExtension(directive string, jazon dynamic.JSON) (any, error) {
	// TODO: register aliases
	// TODO: should separate concerns and wrap another map extension dedicated to directived not related to naming (e.g. x-go-nullable, x-go-type)
	switch directive {
	case "x-go-name", "x-go-package", "x-go-file-name":
		ext := jazon.Interface()
		asString, ok := ext.(string)
		if !ok {
			return nil, fmt.Errorf("invalid %s directive: argument should be a string, but got %T", directive, ext) // TODO: add line number etc
		}
		return asString, nil

	case "x-go-wants-validation", "x-go-validation", "x-go-wants-split-validation", "x-go-split-validation", "x-go-wants-test", "x-go-test":
		ext := jazon.Interface()
		asBool, ok := ext.(bool)
		if !ok {
			return nil, fmt.Errorf("invalid %s directive: argument should be a bool, but got %T", directive, ext) // TODO: add line number etc
		}
		return asBool, nil
	case "x-go-enums":
		ext := jazon.Interface()
		asSlice, ok := ext.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid %s directive: argument should be a slice, but got %T", directive, ext) // TODO: add line number etc
		}
		output := make([]string, 0, len(asSlice))
		for _, elem := range asSlice {
			asString, isString := elem.(string)
			if !isString {
				return nil, fmt.Errorf("invalid %s directive: element in slice should be a string, but got %T", directive, elem) // TODO: add line number etc
			}
			output = append(output, asString)
		}
		return output, nil
	// TODO: x-go-type, x-go-nullable, x-nullable ...
	default:
		return jazon.Interface(), nil // keep directive, but don't perform any check or conversion
	}
}
