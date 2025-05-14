package jsonpath

import (
	"github.com/PaesslerAG/jsonpath"
	"github.com/fredbi/core/json"
)

type PathFinder struct {
	*expressionCache
}

func New() *PathFinder {
	return &PathFinder{}
}

type Expression string

func (p *PathFinder) Get(expr Expression, doc json.Document) []json.Document {

	return nil // TODO
}

func (p *PathFinder) get(rawExpr string, value any) (any, error) {
	return jsonpath.Get(rawExpr, value)
}

type expressionCache struct{}
