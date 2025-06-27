package jsonschema

import (
	"io"
	"iter"
	"slices"

	"github.com/fredbi/core/json/stores"
)

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

func (c *Collection) Schemas() iter.Seq[Schema] {
	return slices.Values(c.schemas)
}

func (c *Collection) Schema(index int) Schema {
	return c.schemas[index]
}

func (c *Collection) Reset() {
	c.schemas = c.schemas[:0]
}

// DecodeAppend decodes JSON from an [io.Reader] an appends it to the [Collection] of schemas.
//
// If the input JSON is an array of schemas, the collection will contain several schemas.
//
// If the input is a JSON schema object, a single schema will be appended.
//
// JSON boolean values "true" and "false" are also valid for JSON schemas.
func (c *Collection) DecodeAppend(reader io.Reader) error {
	sch := Make(withOptions(c.options))
	if err := sch.Decode(reader); err != nil {
		return err
	}
	c.schemas = append(c.schemas, sch)

	return nil
}
