package jsonschema

import (
	"io"
	"iter"
	"slices"

	"github.com/fredbi/core/json/stores"
)

// Collection holds a collection of [Schema] s that share the same document settings (store, etc.).
type Collection struct {
	*options
	schemas []Schema
}

// MakeCollection builds a new schema [Collection] with preallocated capacity cap.
func MakeCollection(cap int, opts ...Option) Collection {
	o := optionsWithDefaults(opts)

	return Collection{
		options: o,
		schemas: make([]Schema, 0, cap),
	}
}

// CollectionFromTemplate creates a new schema [Collection] using a template [Collection].
//
// The new empty collection uses the same set of options as the template
// and pre-allocates the length of the template [Collection].
func CollectionFromTemplate(c Collection) Collection {
	return Collection{
		options: c.options,
		schemas: make([]Schema, 0, c.Len()),
	}
}

// Len returns the current size of the [Collection].
func (c Collection) Len() int {
	return len(c.schemas)
}

func (c *Collection) Store() stores.Store {
	if len(c.schemas) == 0 {
		return nil
	}
	return c.schemas[0].Store()
}

func (c *Collection) Append(schema Schema) {
	c.schemas = append(c.schemas, schema)
}

// Schemas yields an iterator over all [Schema] s in this [Collection].
func (c *Collection) Schemas() iter.Seq[Schema] {
	return slices.Values(c.schemas)
}

func (c *Collection) Schema(index int) Schema {
	return c.schemas[index]
}

func (c *Collection) Reset() {
	c.schemas = c.schemas[:0]
	// TODO: reset options
}

// DecodeAppend decodes a JSON bytes stream from an [io.Reader] an appends it to the [Collection] of schemas.
//
// If the input JSON is an array of schemas, the collection will contain several schemas (TODO).
//
// If the input is a JSON schema object, a single schema will be appended.
//
// Notice that JSON boolean values "true" and "false" are valid JSON schemas.
func (c *Collection) DecodeAppend(reader io.Reader) error {
	sch := Make(withOptions(c.options))
	if err := sch.Decode(reader); err != nil {
		return err
	}
	c.schemas = append(c.schemas, sch)

	return nil
}
