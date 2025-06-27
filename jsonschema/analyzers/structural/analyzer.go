package structural

import (
	"bytes"
	"errors"
	"io"
	"iter"

	"github.com/fredbi/core/jsonschema"
	"github.com/fredbi/core/jsonschema/analyzers"
	"github.com/fredbi/core/jsonschema/analyzers/internal/graph/v2"
	"github.com/fredbi/core/jsonschema/analyzers/structural/order"
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
// Then we get 2 packages defined:
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

	// indexes
	schemas  *schemasGraph // all discovered schemas
	packages *packageTree  // all discovered packages
	// redundant namespaces map[string]namespace                                     // all namespaces
	// redundant contexts   map[graph.Edge[analyzers.UniqueID]]AnalyzedSchemaContext // all dependency contexts (e.g. this schema is used as property etc)
	// redundant readOrder  *list.List                                               // the original ordering of analyzed schemas when read from the json schema (used when marshaling, dumping)

	// graphs
	// schemaGraph *graph.DiGraph[analyzers.UniqueID] // dependency graph of schema. TODO: find a nice way to deal with cyclic deps
	// packageTree *graph.Tree[analyzers.UniqueID]    //  tree of the discovered packages

	validationsAnalyzer *validations.Analyzer // inner analyzer for validations (e.g. when validating zero value, enum value, default value)
}

// NewAnalyzer builds a [SchemaAnalyzer] ready to analyze JSON schemas.
func NewAnalyzer(opts ...Option) *SchemaAnalyzer {
	o = applyOptionsWithDefaults(opts)

	a := &SchemaAnalyzer{
		options:  applyOptionsWithDefaults(opts),
		schemas:  newSchemaGraph(),
		packages: newPackageTree(),
		// contexts:    make(map[graph.Edge[analyzers.UniqueID]]AnalyzedSchemaContext),
		// readOrder:   list.New(),
		// namespaces:  make(map[string]namespace),
		// schemaGraph: graph.NewDiGraph[analyzers.UniqueID](),
		// packageTree: graph.NewTree[analyzers.UniqueID](),
		validationsAnalyzer: validations.New(o.validationOptions...),
	}

	// apply other defaults
	if len(a.extensionMappers) == 0 {
		a.extensionMappers = []ExtensionMapper{passThroughMapper}
	}

	return a
}

func (a *SchemaAnalyzer) SchemaByID(id analyzers.UniqueID) (AnalyzedSchema, bool) {
	return a.schemas.SchemaByID(id)
}

func (a *SchemaAnalyzer) PackageByID(id analyzers.UniqueID) (AnalyzedPackage, bool) {
	return a.packages.PackageByID(id)
}

func (a *SchemaAnalyzer) PackagePaths(filters ...Filter) iter.Seq[string] {
	return typeutils.TransformIter(a.Packages(filters...), func(pkg AnalyzedPackage) (string, bool) {
		return pkg.Path(), true
	})
}

func (a *SchemaAnalyzer) orderedPackageIterator(f filters) iter.Seq[AnalyzedPackage] {
	if f.WantsOnlyLeaves {
		return a.packages.Leaves()
	}

	switch f.Ordering {

	case order.BottomUp:
		return a.packages.Inverted().TraverseBFS()

	case order.TopDown:
		fallthrough

	case order.NoOrder:
		fallthrough
	default:
		return a.packages.TraverseDFS()
	}
}

func (a *SchemaAnalyzer) Packages(filterSpecs ...Filter) iter.Seq[AnalyzedPackage] {
	filter := applyFiltersWithDefault(filterSpecs)
	iterator := a.orderedPackageIterator(filter)

	if filter.PkgFilterFunc != nil {
		iterator = typeutils.FilterIter(iterator, filter.PkgFilterFunc)
	}

	// TODO: should add index, required index
	return iterator
}

// Analyze a single JSON schema.
func (a *SchemaAnalyzer) Analyze(schema jsonschema.Schema) error {
	// 1. package analysis
	for analyzed := range a.exploreSchemas(schema) { // all analyzed schemas, depth-first
		schemaNode, err := a.schemas.AddArc(analyzed)
		//edge := a.schemas.NewEdge(analyzed, analyzed, analyzed.Context())
		if err != nil {
			if errors.Is(graph.ErrCycleFound) {
				// a cycle is found : keep the cycle for later processing
				//a.schema.AddCycle()
			}

			return err // should not happen
		}

		pth := analyzed.Path()
		if pth == "" {
			continue
		}

		// found a path: add package
		packageNode, found := a.packages.AddPath(pth)

		if err = packageNode.AddDependency(analyzed); err != nil {
			// a cycle is found in dependencies: keep the package-level cycle for later refact
		}

		if found {
			continue
		}
	}

	// 2. package refactoring : may invoke callback
	if err := a.refactorPackages(); err != nil {
		// TODO: log / audit
		return err
	}

	// now packages form a tree and packagesDependencies is a DAG

	// 3. schema cycles processing
	if err := a.analyzeSchemaCycles(); err != nil {
		// TODO: log / audit
		return err
	}

	// 4. schema refactoring actions
	if err := a.refactorSchemas(); err != nil {
		// TODO: log / audit
		return err
	}

	// 5. schema naming callbacks
	if err := a.visitSchemas(); err != nil {
		// TODO: log / audit
		return err
	}

	return nil
}

func (a *SchemaAnalyzer) refPath(schema jsonschema.Schema) string {
	return ""
}

func (a *SchemaAnalyzer) analyzeMetadata(schema jsonschema.Schema, analyzed *AnalyzedSchema) {
	if !schema.HasMetadata() {
		return
	}

	meta := Metadata{
		//ID:
		// Path:
		Metadata: schema.Metadata(),
	}
	analyzed.meta = meta
}

func (a *SchemaAnalyzer) refIsAlreadyAnalyzed(schema jsonschema.Schema) bool {
	if !schema.HasRef() {
		return false
	}

	ref := schema.Ref()
	_, found := a.schemas.SchemaByPath(ref.String())

	return found
}

func (a *SchemaAnalyzer) exploreSchemas(schema jsonschema.Schema) iter.Seq[AnalyzedSchema] {
	return func(yield func(AnalyzedSchema) bool) {
		// TODO: options if we want to explore unused stuff (e.g. subtypes in allOf)
		var analyzed AnalyzedSchema

		// TODO: do we have it already

		// TODO handle extensions

		if a.refIsAlreadyAnalyzed(schema) {
			if !yield(analyzed) {
				return
			}
		} else {
			if schema.HasRef()

			for subschema := range a.exploreSchemas(refSchema) {
				if !yield(subschema) {
					return
				}
			}
		}

		a.analyzeMetadata(schema, &analyzed)

		for propertyName, propertySchema := range schema.Properties() {
			if !yield(analyzed) {
				return
			}
		}

		for propertyName, propertySchema := range schema.PatternProperties() {
			if !yield(analyzed) {
				return
			}
		}

		for propertyName, propertySchema := range schema.DependentSchemas() {
			if !yield(analyzed) {
				return
			}
		}

		if schema.HasAdditionalProperties() {
			if !yield(analyzed) {
				return
			}
		}

		for index, itemsSchema := range schema.TupleItems() {
			if !yield(analyzed) {
				return
			}
		}

		if schema.HasTupleAdditionalItems() {
			if !yield(analyzed) {
				return
			}
		}

		for index, allOfSchema := range schema.AllOf() {
			if !yield(analyzed) {
				return
			}
		}

		for index, allOfSchema := range schema.AnyOf() {
			if !yield(analyzed) {
				return
			}
		}

		for index, allOfSchema := range schema.OneOf() {
			if !yield(analyzed) {
				return
			}
		}
	}
}

// Analyze a collection of JSON schemas to reason about their structure.
func (a *SchemaAnalyzer) AnalyzeCollection(schemas jsonschema.Collection) error {
	// TODO: merge collection
	for schema := range schemas.Schemas() {
		if err := a.Analyze(schema); err != nil {
			return err
		}
	}

	return nil
}

func (a *SchemaAnalyzer) orderedSchemaIterator(f filters) iter.Seq[AnalyzedSchema] {
	if f.WantsOnlyLeaves {
		return a.schemas.Leaves()
	}

	switch f.Ordering {

	case order.BottomUp:
		return a.schemas.Inverted().TraverseTopological()

	case order.TopDown:
		return a.schemas.TraverseDFS()

	case order.NoOrder:
		fallthrough
	default:
		return a.schemas.Nodes()
	}
}

// AnalyzedSchemas yields the analyzed schemas according to some filter expression.
func (a *SchemaAnalyzer) AnalyzedSchemas(filterSpecs ...Filter) iter.Seq[AnalyzedSchema] {
	filter := applyFiltersWithDefault(filterSpecs)
	iterator := a.orderedSchemaIterator(filter)

	if filter.FilterFunc != nil {
		iterator = typeutils.FilterIter(iterator, filter.FilterFunc)
	}

	// TODO: add index
	return iterator
}

// Len indicates how many unitary schemas are held by the analyzer.
func (a *SchemaAnalyzer) Len() int {
	return a.schemas.Len()
}

func (a *SchemaAnalyzer) Encode(w io.Writer) error {
	// TODO: options to remove extensions, etc.
	for node := range a.schemas.Nodes() {
		schema := node.document
		if err := schema.document.Encode(&w); err != nil {
			return nil, err
		}
	}

	return nil
}

func (a *SchemaAnalyzer) MarshalJSON() ([]byte, error) {
	var w bytes.Buffer

	if err := a.Encode(&w); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// Dump writes out the analyzed JSON schema.
func (a *SchemaAnalyzer) Dump(w io.Writer) error {
	return a.Encode(w)
}
