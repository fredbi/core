package structural

import (
	"errors"
	"io"
	"iter"

	"github.com/fredbi/core/jsonschema"
	"github.com/fredbi/core/jsonschema/analyzers"
)

// Analyzer is the interface for a structural analyzer.
//
// It allows consuming packages to build mocks.
type Analyzer interface {
	// Analyze a single JSON schema
	Analyze(jsonschema.Schema) error

	// Analyze an entire collection of JSON schemas
	AnalyzeCollection(jsonschema.Collection) error

	// AnalyzedSchemas iterates over the analyzed schemas, possibly applying some [Filter]
	AnalyzedSchemas(...Filter) iter.Seq[AnalyzedSchema]

	// Number of analyzed schemas, with all inner sub-schemas
	Len() int

	// Bundle the collection of analyzed schemas, reorganizing the namespace.
	Bundle(...BundleOption) (Analyzer, error)

	// Namespaces iterates over the namespace of bundled package names
	Namespaces(...Filter) iter.Seq[string]

	// Packages iterates over the namespace of bundled packages
	Packages(...Filter) iter.Seq[AnalyzedPackage]

	// SchemaByID yields a single schema, given its unique key ID.
	SchemaByID(analyzers.UniqueID) (AnalyzedSchema, bool)
}

var _ Analyzer = &SchemaAnalyzer{}

// SchemaAnalyzer knows how to analyze the structure of a JSON schema specification to generate artifacts.
type SchemaAnalyzer struct {
	options
	index      map[analyzers.UniqueID]*AnalyzedSchema
	forest     []AnalyzedSchema // TODO: dependency graph
	namespaces map[string]Namespace
	packages   []AnalyzedPackage
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

// Bundle reforms a new analyzer by bundling references, optionnally applying namespace and naming rules.
func (a *SchemaAnalyzer) Bundle(_ ...BundleOption) (Analyzer, error) {
	return nil, errors.New("not implemented") // TODO
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
