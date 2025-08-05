package jsonpath

import (
	"iter"

	//"github.com/PaesslerAG/jsonpath"
	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/dynamic"
)

// PathFinder resolves a JSONPath [Expression] s against a [json.Document] or a [dynamic.JSON] structure.
type PathFinder struct {
	*expressionCache
}

func New() *PathFinder {
	return &PathFinder{}
}

// Expression is a JSONPath expression.
type Expression struct {
	text []byte
}

type StringOrBytes interface {
	string | []byte
}

func MakeExpression[T StringOrBytes](jp T) (Expression, error) {
	e := Expression{
		text: []byte(jp),
	}
	// TODO: parse JSONPath expression

	return e, nil
}

func NewExpression[T StringOrBytes](jp T) (*Expression, error) {
	e, err := MakeExpression(jp)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func (e Expression) String() string {
	return string(e.text)
}

func (p *PathFinder) Get(root json.Document, expr Expression) iter.Seq[json.Document] {
	return nil // TODO
}

func (p *PathFinder) GetDynamic(root dynamic.JSON, expr Expression) iter.Seq[dynamic.JSON] {
	return nil // TODO
}

func (p *PathFinder) Pointers(root json.Document, expr Expression) iter.Seq[json.Pointer] {
	return nil // TODO
}

/*
func (p *PathFinder) get(rawExpr string, value any) (any, error) {
	_ = rawExpr
	_ = value
	//return jsonpath.Get(rawExpr, value)
	return nil, nil
}
*/

type expressionCache struct{}
