package jsonschema

import "github.com/fredbi/core/json"

type SchemaVersion uint8

const (
	SchemaVersionDraft4 SchemaVersion = iota
	SchemaVersionDraft6
	SchemaVersionDraft7
	SchemaVersionDraft2019
	SchemaVersionDraft2020
)

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
