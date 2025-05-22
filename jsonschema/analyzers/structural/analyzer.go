package structural

import (
	"iter"

	"github.com/fredbi/core/jsonschema"
)

type AnalyzedSchema struct {
	IsNamed bool
	Name    string
	//Ref
	RefLocation string // $ref path
	Tag         string // x-go-tag
}

// Analyzer knows how to analyze the structure of a JSON schema specification
// to generate artifacts.
type Analyzer struct {
	forest []AnalyzedSchema // TODO: dependency graph
}

type Option func(*options)

type options struct{}

func New(_ ...Option) *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) Analyze(jsonschema.Schema) error {
	return nil // TODO
}

func (a *Analyzer) AnalyzeCollection(jsonschema.SchemaCollection) error {
	return nil // TODO
}

// BottomUpSchemas iterates over the dependency graph of schemas from leave nodes up to the root nodes.
func (a *Analyzer) BottomUpSchemas(...Filter) iter.Seq[AnalyzedSchema] {
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
func (a *Analyzer) Len() int {
	return len(a.forest) // TODO
}

type Filter func(f *filters)

type filters struct {
	WantsNamed bool
}

func OnlyNamedSchemas() Filter {
	return func(f *filters) {
		f.WantsNamed = true
	}
}
