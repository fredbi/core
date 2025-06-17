package validations

import (
	"github.com/fredbi/core/json"
	"github.com/fredbi/core/jsonschema"
	"github.com/fredbi/core/jsonschema/analyzers/canonical/ast"
)

type DocumentValidatorFunc func(json.Document) error
type JSONValidatorFunc func([]byte) error

type Analyzer struct {
	tree ast.Tree
}

type AnalyzedSchema struct {
}

type Option func(*options)

type options struct{}

func New(_ ...Option) *Analyzer {
	return &Analyzer{}
}

type CollectionAnalyzer struct {
	forest ast.Forest
}

func (a *Analyzer) Analyze(_ jsonschema.Schema) error {
	return nil
}

func (a *Analyzer) AnalyzeCollection(_ jsonschema.Collection) error {
	return nil
}

func (a *Analyzer) DocumentValidator() DocumentValidatorFunc {
	return nil
}

func (a *Analyzer) JSONValidator() JSONValidatorFunc {
	return nil
}

func (a *Analyzer) AST() *ast.Tree {
	return nil
}

func (a *Analyzer) CanonicalSchema() jsonschema.Schema {
	return jsonschema.Schema{}
}
