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
//
// You should always check the final error state, since the building ceases to be effective
// as soon as an error is encountered.
type Builder struct {
	doc         Document
	nodeBuilder *light.Builder
}

// NewBuilder produces a [Builder] of JSON [Document] s.
func NewBuilder(s stores.Store) *Builder {
	b := &Builder{
		doc:         EmptyDocument,
		nodeBuilder: light.NewBuilder(s),
	}

	b.doc.store = s

	return b
}

func (b Builder) Err() error {
	return b.nodeBuilder.Err()
}

func (b Builder) Ok() bool {
	return b.nodeBuilder.Ok()
}

func (b Builder) Store() stores.Store {
	return b.doc.store
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
	b.nodeBuilder = b.nodeBuilder.WithStore(s)

	return b
}

// MakeNull is a shorthand for NewBuilder(b.Store()).Null().Document().
func (b *Builder) MakeNull() Document {
	return NewBuilder(b.Store()).Null().Document()
}

// MakeBool is a shorthand for NewBuilder(b.Store()).BoolValue(value).Document().
func (b *Builder) MakeBool(value bool) Document {
	return NewBuilder(b.Store()).BoolValue(value).Document()
}

// MakeString is a shorthand for NewBuilder(b.Store()).StringValue(value).Document().
func (b *Builder) MakeString(value string) Document {
	return NewBuilder(b.Store()).StringValue(value).Document()
}

// MakeNumber is a shorthand for NewBuilder(b.Store()).NumericalValue(value).Document().
func (b *Builder) MakeNumber(value any) Document {
	v := NewBuilder(b.Store()).NumericalValue(value)
	if v.Ok() {
		return v.Document()
	}

	b.SetErr(v.Err())

	return v.Document()
}

// Document returns the [Document] produced by the [Builder].
//
// If a build error has occurred, it returns the [EmptyDocument].
func (b Builder) Document() Document {
	if !b.Ok() {
		return EmptyDocument
	}

	return b.doc
}

// From makes a builder that will clone a [Document], possibly mutations.
func (b *Builder) From(d Document) *Builder {
	b.doc = d
	b.nodeBuilder.Reset()

	return b
}

func (b *Builder) WithRoot(root light.Node) *Builder {
	b.doc.root = root
	b.nodeBuilder.Reset()

	return b
}

// Object builds a JSON object.
func (b *Builder) Object() *Builder {
	bn := b.nodeBuilder
	bn.Reset()
	b.doc.root = bn.Object().Node()

	return b
}

// Array builds a JSON array.
func (b *Builder) Array() *Builder {
	bn := b.nodeBuilder
	bn.Reset()
	b.doc.root = bn.Array().Node()

	return b
}

// AppendKey appends a new (key,value) to an object.
func (b *Builder) AppendKey(key string, value Document) *Builder {
	bn := b.nodeBuilder
	bn.Reset()
	b.doc.root = bn.From(b.doc.root).AppendKey(key, value.root).Node()

	return b
}

// AppendElem appends a new element to an array.
func (b *Builder) AppendElem(value Document) *Builder {
	bn := b.nodeBuilder
	bn.Reset()
	b.doc.root = bn.From(b.doc.root).AppendElem(value.root).Node()

	return b
}

// AppendElems appends new elements to an array.
func (b *Builder) AppendElems(values ...Document) *Builder {
	if !b.Ok() {
		return b
	}

	for _, value := range values {
		_ = b.AppendElem(value)
		if !b.Ok() {
			break
		}
	}

	return b
}

// StringValue builds a scalar JSON string
func (b *Builder) StringValue(value string) *Builder {
	bn := b.nodeBuilder
	bn.Reset()
	b.doc.root = bn.StringValue(value).Node()

	return b
}

// BoolValue builds a scalar JSON boolean value.
func (b *Builder) BoolValue(value bool) *Builder {
	bn := b.nodeBuilder
	bn.Reset()
	b.doc.root = bn.BoolValue(value).Node()

	return b
}

// NumberValue builds a scalar JSON number value.
func (b *Builder) NumberValue(value types.Number) *Builder {
	bn := b.nodeBuilder
	bn.Reset()
	b.doc.root = bn.NumberValue(value).Node()

	return b
}

// NumericalValue builds a scalar JSON number value from any go numerical type, including types from math/big.
func (b *Builder) NumericalValue(value any) *Builder {
	bn := b.nodeBuilder
	bn.Reset()
	b.doc.root = bn.NumericalValue(value).Node()

	return b
}

// NullValue builds a scalar JSON null value.
func (b *Builder) Null() *Builder {
	bn := b.nodeBuilder
	bn.Reset()
	b.doc.root = bn.Null().Node()

	return b
}

// TODO: pick all build methods from light.Node

// AtPointer replaces a value in a [Document] at [Pointer].
//
// No replacement is made if the [Pointer] is not found.
func (b *Builder) AtPointer(p Pointer, value Document) *Builder {
	/* TODO
	resolved, err := b.doc.setNodePointer(&b.doc.root, p, value)
	if err != nil {
		// silently ignores unresolved pointers
		return b
	}
	*/

	return b
}

func (b *Builder) AtPointerMerge(p Pointer, value Document) *Builder {
	// TODO
	return b
}
