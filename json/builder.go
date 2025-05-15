package json

import (
	"github.com/fredbi/core/json/nodes/light"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/types"
)

// Builder builds or transforms JSON [Document] s programmatically.
//
// You may either use it directly, starting from the [EmptyDocument] or clone from an existing [Document] using
// [Builder.From].
//
// Since a [Document] is immutable, the [Builder] always produces a shallow clone of the original [Document].
//
// The [Builder] exposes fluent building methods which may be chained to construct a JSON document.
// You should always check the final error state, since the building cease to be effective
// as soon as an error is encountered.
type Builder struct {
	doc         Document
	nodeBuilder light.Builder
}

func MakeBuilder(s stores.Store) Builder {
	b := Builder{
		doc: EmptyDocument,
	}

	b.doc.store = s

	return b
}

func NewBuilder(s stores.Store) *Builder {
	b := MakeBuilder(s)

	return &b
}

func (b Builder) Err() error {
	return b.nodeBuilder.Err()
}

func (b Builder) Ok() bool {
	return b.nodeBuilder.Ok()
}

func (b *Builder) SetErr(err error) {
	b.nodeBuilder.SetErr(err)
}

func (b *Builder) Reset() {
	b.doc = EmptyDocument
	b.nodeBuilder.Reset()
}

func (b *Builder) WithStore(s stores.Store) *Builder {
	b.doc.store = s

	return b
}

// Document returns the [Document] produced by the [Builder].
//
// If a build error has occured, it returns the [EmptyDocument].
func (b Builder) Document() Document {
	if !b.Ok() {
		return EmptyDocument
	}

	return b.doc
}

func (b *Builder) From(d Document) *Builder {
	b.doc = d
	b.nodeBuilder.Reset()

	return b
}

// Object builds a JSON object
func (b *Builder) Object() *Builder {
	bn := b.nodeBuilder
	bn.Reset()
	b.doc.root = bn.Object().Node()

	return b
}

// Array builds a JSON array
func (b *Builder) Array() *Builder {
	bn := b.nodeBuilder
	bn.Reset()
	b.doc.root = bn.Array().Node()

	return b
}

func (b *Builder) AppendKey(key string, value Document) *Builder {
	bn := b.nodeBuilder
	bn.Reset()
	b.doc.root = bn.From(b.doc.root).AppendKey(key, value.root).Node()

	return b
}

func (b *Builder) AppendElem(value Document) *Builder {
	bn := b.nodeBuilder
	bn.Reset()
	b.doc.root = bn.From(b.doc.root).AppendElem(value.root).Node()

	return b
}

// StringValue builds a JSON string
func (b *Builder) StringValue(value string) *Builder {
	bn := b.nodeBuilder
	bn.Reset()
	b.doc.root = bn.StringValue(value).Node()

	return b
}

func (b *Builder) BoolValue(value bool) *Builder {
	// TODO
	return b
}

func (b *Builder) FloatValue(value types.Number) *Builder {
	// TODO

	return b
}

func (b *Builder) NumberValue(value types.Number) *Builder {
	// TODO
	return b
}

func (b *Builder) Null() *Builder {
	// TODO
	return b
}
