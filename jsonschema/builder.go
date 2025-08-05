package jsonschema

import (
	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/nodes/light"
)

// Builder constructs a [Schema].
//
// TODO
type Builder struct {
	err error
	sch Schema
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b Builder) Err() error {
	return b.err
}

func (b Builder) Ok() bool {
	return b.err == nil
}

func (b Builder) Schema() Schema {
	return b.sch
}

func (b *Builder) From(sch Schema) *Builder {
	b.sch = sch

	return b
}

func (b *Builder) WithRoot(root light.Node) *Builder {
	// TODO
	return b
}

// AtPointer replaces a value at the location pointed at.
func (b *Builder) AtPointer(p json.Pointer, value json.Document) *Builder {
	return b // TODO
}

func (b *Builder) AtPointerMerge(p json.Pointer, value json.Document) *Builder {
	return b // TODO
}

func (b *Builder) WithProperties(properties []Schema) *Builder {
	return b // TODO
}

func (b *Builder) WithTypes(types []string) *Builder {
	return b // TODO
}

func (b *Builder) WithAllOf(schemas ...Schema) *Builder {
	return b // TODO
}

func (b *Builder) WithOneOf(schemas ...Schema) *Builder {
	return b // TODO
}

func (b *Builder) WithAnyOf(schemas ...Schema) *Builder {
	return b // TODO
}
