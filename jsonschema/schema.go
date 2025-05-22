package jsonschema

import "github.com/fredbi/core/json"

type Schema struct {
	json.Document

	// decoded syntax
}

func Make(opts ...Option) Schema {
	return Schema{} // TODO
}

func New(opts ...Option) *Schema {
	s := Make(opts...)

	return &s
}

type SchemaCollection struct {
	schemas []Schema
}

func (c SchemaCollection) Len() int {
	return len(c.schemas)
}

type NamedSchema struct {
	Key string
	Schema
}
