package analyzer

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

func New( /* opts ...Option */ ) *Analyzer {
	return &Analyzer{}
}

type CollectionAnalyzer struct {
	forest ast.Forest
}

func (a *Analyzer) Analyze(s jsonschema.Schema) error {
	return nil
}

func (a *Analyzer) DocumentValidator() DocumentValidatorFunc {
	return nil
}

func (a *Analyzer) JSONValidator() JSONValidatorFunc {
	return nil
}

func (a *Analyzer) AST() ast.Tree {
	return nil
(

func (a *Analyzer) CanonicalSchema() jsonschema.Schema {
	return jsonschema.Schema{}
}
