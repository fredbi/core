package structural

import (
	"errors"
	"io"
	"iter"

	"github.com/fredbi/core/jsonschema"
	"github.com/fredbi/core/jsonschema/analyzers"
	"github.com/fredbi/core/jsonschema/analyzers/validations"
	"github.com/fredbi/core/swag/typeutils"
)

var _ Analyzer = &SchemaAnalyzer{}

// SchemaAnalyzer knows how to analyze the structure of a JSON schema specification to generate artifacts.
//
// A [SchemaAnalyzer] analyzes JSON schemas so we can reason about their structure.
//
// The difference with a [validations.Analyzer] lies in the focus on finding correct ways to serialize a JSON schema.
//
// # Namepace analysis
//
// The analysis isolates two different aspects of the schema specification:
//
//   - packages (created by the use of '$ref' and '$id')
//   - schemas
//
// Schemas that are referred to by a '$ref', are included in '$defs' (or "definition") or have an '$id'
// are deemed "named schemas". Other schemas are anonymous.
//
// A "package" is defined by the base URL of a '$ref' or '$id'.
//
// Example:
//
// If we have 3 named schemas like so:
//
//	sch1:
//	  $ref: #/$defs/named
//	sch2:
//	  $ref: #/$defs/named
//	sch3:
//	  $ref: #customers/models.json/$defs/named
//
//	$defs:
//	  named:
//	    {...}
//
// We have 2 packages defined:
//
//   - the first one in the '$defs' of the root document
//   - the other one in 'customers/models.json/$defs'
//
// During a bundling operation, all visited packages and schemas may be renamed or assigned a name with provided optional
// callbacks.
//
// Anonymous schemas that are assigned a name will be relocated under the nearest '$defs' object.
//
// # Dependencies analysis
//
// All analyzed schemas are organized as a graph of dependencies that may be iterated over in different ways using a
// [order.SchemaOrdering] clause in a [Filter].
//
// # Schema refactoring
//
// Given the appropriate [Option] s, the outcome of the analysis may be a refactored schema, with an equivalent
// validation outcome, but with a structure that is more amenable to a serialized specification.
//
// One of the goals of refactoring is to reduce the variability of how to express the same validations with only
// a subset of JSON schema grammar.
//
// In some cases, such as 'const', the differencee is only lexical and doesn't really a transformation ('const' is
// just a 'enum' with one value). In other situations like combinations of 'allOf', 'anyOf', 'oneOf', 'not', 'if',
// 'then', 'else', we may rewrite the schema so consumers of the [AnalyzedSchema] may take simpler assumptions.
//
// Typical refactoring actions include the reorganization of compositions like 'allOf', 'anyOf', 'oneOf'.
//
// Supported refactoring actions:
//
//   - reduce schemas that always evaluate to true or false
//   - lift anonymous 'allOf' members
//   - prune enum values that do not match other validations
//   - split multiple 'type' arrays into 'oneOf' or 'anyOf', ensure there is always 1 and 1 only type (except when any).
//     Doing so, validations that do not apply to the type are pruned
//   - push 'allOf' for arrays down to the 'items' level
//   - transform compositions 'allOf', 'anyOf', 'oneOf' so only one is present at a schema level
//   - lift compositions that reduce to one anonymous member only
//   - split overlapping properties in 'allOf'
//   - rewrite 'if', 'then', 'else' validations with 'oneOf', 'allOf' and 'not'.
//
// # Schema bundling
//
// Bundling transforms a schema or collection of schemas into a single document with no remote '$ref's, so the resulting
// JSON schema document is self-contained.
//
// All named schemas are placed in '$defs' definitions.
//
// The bundling process may adopt different strategies to organize the namepace of named schemas with no name conflicts.
//
// The default strategy is to organize '$defs' into a hierarchy following the declared namespaces.
// This strategy is conflict-free.
//
// Alternative strategies may be used to flatten the namespace, or use an hybrid approach.
//
// In addition, bundling may optionally refactor enums validations to define a sub-package for these declarations
// (see [WithBundleEnumPackage]).
//
// # Extensions
//
// The [SchemaAnalyzer] doesn't interpret or use OpenAPI-style extensions ("x-*).
//
// However, it may invoke extension processing callbacks whenever it encounters one.
//
// This is applied before any other callbacks are invoked on a schema.
//
// Exception:
//   - the [SchemaAnalyzer] produces a specific extension to report about its transformations in an audit trail
//     (see below): "x-go-audit"
//
// # Auditability
//
// Refactoring and bundling actions on the original schema are tracked by an audit trail that may be consumed
// in the [AnalyzedSchema].
//
// Decisions that are taken by external callback may also be tracked: the callbacks have to call
// [SchemaAnalyzer.LogAudit] to document their action.
type SchemaAnalyzer struct {
	options
	bundleOptions

	index    map[analyzers.UniqueID]*AnalyzedSchema
	pkgIndex map[analyzers.UniqueID]*AnalyzedPackage

	forest              []AnalyzedSchema // TODO: dependency graph
	namespaces          map[string]Namespace
	packages            []AnalyzedPackage
	validationsAnalyzer *validations.Analyzer
}

// NewAnalyzer builds a [SchemaAnalyzer] ready to analyze JSON schemas.
func NewAnalyzer(opts ...Option) *SchemaAnalyzer {
	a := &SchemaAnalyzer{
		options: applyOptionsWithDefaults(opts),
	}

	if len(a.extensionMappers) == 0 {
		a.extensionMappers = []ExtensionMapper{passThroughMapper}
	}

	return a
}

func (a *SchemaAnalyzer) SchemaByID(id analyzers.UniqueID) (AnalyzedSchema, bool) {
	schema, ok := a.index[id]

	return *schema, ok
}

func (a *SchemaAnalyzer) Namespaces(filters ...Filter) iter.Seq[string] {
	return func(yield func(string) bool) {
		for key := range a.namespaces {
			// todo apply filters
			if !yield(key) {
				return
			}
		}
	}
}

func (a *SchemaAnalyzer) Packages(filters ...Filter) iter.Seq[AnalyzedPackage] {
	return func(yield func(AnalyzedPackage) bool) {
		for _, node := range a.packages {
			// todo apply filters
			if !yield(node) {
				return
			}
		}
	}

}

// Analyze a single JSON schema.
func (a *SchemaAnalyzer) Analyze(jsonschema.Schema) error {
	return nil // TODO
}

// Analyze a collection of JSON schemas to reason about their structure.
func (a *SchemaAnalyzer) AnalyzeCollection(jsonschema.Collection) error {
	return nil // TODO
}

// AnalyzedSchemas yields the analyzed schemas according to some filter expression.
func (a *SchemaAnalyzer) AnalyzedSchemas(...Filter) iter.Seq[AnalyzedSchema] {
	return func(yield func(AnalyzedSchema) bool) {
		for _, node := range a.forest {
			// todo apply filters
			if !yield(node) {
				return
			}
		}
	}
}

// Len indicates how many unitary schemas are held by the analyzer.
func (a *SchemaAnalyzer) Len() int {
	return len(a.forest) // TODO
}

func (a *SchemaAnalyzer) MarshalJSON() ([]byte, error) {
	return nil, errors.New("not implemented")
}

// Dump writes out the analyzed JSON schema.
func (a *SchemaAnalyzer) Dump(w io.Writer) error {
	content, err := a.MarshalJSON()
	if err != nil {
		return err
	}

	_, err = w.Write(content)

	return err
}

func (a *SchemaAnalyzer) LogAudit(s AnalyzedSchema, e AuditTrailEntry) {
	if e.Action == AuditActionNone {
		return
	}

	schema, found := a.index[s.id]
	if !found {
		return
	}

	schema.auditEntries = append(schema.auditEntries, e)
}

func (a *SchemaAnalyzer) LogAuditPackage(p AnalyzedPackage, e AuditTrailEntry) {
	if e.Action == AuditActionNone {
		return
	}

	pkg, found := a.pkgIndex[p.id]
	if !found {
		return
	}

	pkg.auditEntries = append(pkg.auditEntries, e)
}

func (a *SchemaAnalyzer) MarkSchema(s AnalyzedSchema, e Extensions) {
	if len(e) == 0 {
		return
	}

	schema, found := a.index[s.id]
	if !found {
		return
	}

	schema.extensions = typeutils.MergeMaps(schema.extensions, e)
}
