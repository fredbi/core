package jsonschema

import (
	"io"
	"iter"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/stores"
)

type Schema struct {
	json.Document

	// decoded syntax
	source string
}

func Make(opts ...Option) Schema {
	// TODO: default store
	return Schema{} // TODO
}

func New(opts ...Option) *Schema {
	s := Make(opts...)

	return &s
}

type Collection struct {
	options
	schemas []Schema
}

func MakeCollection(cap int, opts ...Option) Collection {
	return Collection{
		schemas: make([]Schema, 0, cap),
	}
}

func (c Collection) Len() int {
	return len(c.schemas)
}

func (c *Collection) Store() stores.Store {
	return c.store
}

func (c *Collection) Append(schema Schema) {
	c.schemas = append(c.schemas, schema)
}

func (c *Collection) Schemas() iter.Seq2[int, Schema] { // TODO : return iterator
	return nil
}

func (c *Collection) Schema(index int) Schema {
	return c.schemas[index]
}

func (c *Collection) Reset() {
	c.schemas = c.schemas[:0]
}

func (c *Collection) DecodeAppend(reader io.Reader) error {
	sch := Make(withOptions(c.options))
	if err := sch.Decode(reader); err != nil {
		return err
	}
	c.schemas = append(c.schemas, sch)

	return nil
}

type NamedSchema struct {
	Key string
	Schema
}
