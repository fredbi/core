package json

import (
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"iter"

	"github.com/fredbi/core/json/internal"
	"github.com/fredbi/core/json/lexers"
	"github.com/fredbi/core/json/nodes"
	"github.com/fredbi/core/json/nodes/light"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/writers"
)

var (
	_ json.Marshaler        = Document{}
	_ json.Unmarshaler      = &Document{}
	_ encoding.TextAppender = Document{}
)

// EmptyDocument is the JSON document of the null type.
//
// It has no [stores.Store] attached to it.
var EmptyDocument = Make()

// Document represents a JSON document as a hierarchy of JSON data nodes.
//
// A [Document] knows how to marshal or unmarshal bytes or decode/encode with streams.
//
// A [Document] is immutable: it may be unmarshaled from JSON, from [Dynamic JSON] or built programmatically using the [Builder].
//
// # Accessing nodes
//
// [Document.AtKey] retrieves an individual keys in an object. [Document.Elem] does the same for an element in an array.
//
// # Iterators
//
// The hierarchy of nodes defined by a [Document] may be explored using iterators like [Document.Pairs] and
// [Document.Elems].
//
// Iterators maintain the original order in which object keys and array elements have been provided.
//
// # Exploring a document
//
// JSON pointers are supported within a [Document] using [Document.GetPointer].
//
// An implementation of JSONPath is provided in [github.com/fredbi/core/json/documents.jsonpath] to resolve
// JSONPath expressions as a [Document] iterator.
//
// # Working with values
//
// TODO(fred): documentation
//
// # Dynamic JSON
//
// We call "dynamic JSON" (sometimes referred to as "untyped") refers to the go structures made up of "map[string]any"
// and "[]any" that you typically get when the go standard library unmarshals JSON into an "any" type
// (aka "interface{}").
//
// A [Document] may unmarshal such a data structure or may be converted into one.
//
// In that case, due to the implementation of go maps, the order of keys in objects cannot be maintained.
//
// # Related projects
//
// This package only deals with pure JSON, not schemas.
//
// Package [github.com/fredbi/core/jsonschma] brings the additional logic required to deal with JSON schemas.
//
// Package [github.com/fredbi/core/spec] brings the additional logic required to deal with OpenAPI documents.
type Document struct {
	options
	document
}

// Context of a node, i.e. the offset in the originally parsed JSON input.
type Context struct {
	light.Context
}

type document struct {
	root light.Node
}

// Make an empty [Document].
//
// The empty [Document] marshals as "null".
func Make(opts ...Option) Document {
	return Document{
		options: optionsWithDefaults(opts),
	}
}

func (d Document) fromNode(n light.Node) Document {
	return Document{
		options: d.options,
		document: document{
			root: n,
		},
	}
}

func (d Document) Store() stores.Store {
	return d.store
}

// Node low-level access to the current node in the document hierarchy.
func (d Document) Node() light.Node {
	return d.root
}

func (d Document) Context() Context {
	return Context{Context: d.root.Context()}
}

func (d Document) Value() (stores.Value, bool) {
	return d.root.Value(d.store)
}

// AtKey returns the value held under a key in an object, or false if not found.
//
// Key lookup is constant-time (map index).
func (d Document) AtKey(k string) (Document, bool) {
	n, b := d.root.AtKey(k)
	if !b {
		return EmptyDocument, false
	}

	return d.fromNode(n), true
}

// KeyIndex returns the index of a key, of false if not found.
func (d Document) KeyIndex(k string) (int, bool) {
	return d.root.KeyIndex(k)
}

// Elem returns the i-th element of an array.
func (d Document) Elem(i int) (Document, bool) {
	n, b := d.root.Elem(i)
	if !b {
		return EmptyDocument, false
	}

	return d.fromNode(n), true
}

// Pairs return all (key,Node) pairs inside an object.
//
// Iteration order is stable and honors the original ordering
// in which keys were declared.
func (d Document) Pairs() iter.Seq2[string, Document] {
	return func(yield func(string, Document) bool) {
		for _, pair := range d.root.Pairs() {
			if !yield(pair.Key(), d.fromNode(pair)) {
				return
			}
		}
	}
}

// Elems returns all elements in an array.
//
// Iteration order is stable and honors the original ordering
// in which elements were declared.
func (d Document) Elems() iter.Seq[Document] {
	return func(yield func(Document) bool) {
		for node := range d.root.Elems() {
			if !yield(d.fromNode(node)) {
				return
			}
		}
	}
}

func (d Document) Kind() nodes.Kind {
	return d.root.Kind()
}

func (d Document) Len() int {
	return d.root.Len()
}

// Decode builds a [Document] from a stream of JSON bytes.
func (d *Document) Decode(r io.Reader) error {
	lex, redeem := d.lexerFromReaderFactory(r)
	defer redeem()

	return d.decode(lex)
}

// UnmarshalJSON builds a [Document] from JSON bytes.
func (d *Document) UnmarshalJSON(data []byte) error {
	lex, redeem := d.lexerFactory(data)
	defer redeem()

	return d.decode(lex)
}

// Encode the [Document] as a JSON stream to an [io.Writer].
func (d Document) Encode(w io.Writer) error {
	jw, redeem := d.writerToWriterFactory(w)
	defer redeem()

	return d.encode(jw)
}

// AppendText appends the JSON bytes to the provided buffer and returns the resulting slice.
func (d Document) AppendText(b []byte) ([]byte, error) {
	w := internal.BorrowAppendWriter()
	w.Set(b)
	jw, redeem := d.writerToWriterFactory(w)
	defer func() {
		internal.RedeemAppendWriter(w)
		redeem()
	}()

	err := d.encode(jw)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// MarshalJSON writes the [Document] as JSON bytes.
func (d Document) MarshalJSON() ([]byte, error) {
	buf := internal.BorrowBytesBuffer()
	jw, redeem := d.writerToWriterFactory(buf)
	defer func() {
		internal.RedeemBytesBuffer(buf)
		redeem()
	}()

	if err := d.encode(jw); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (d Document) String() string {
	if d.root.Kind() == nodes.KindScalar {
		v, _ := d.root.Value(d.store)
		return v.String()
	}

	buf := internal.BorrowBytesBuffer()
	jw, redeem := d.writerToWriterFactory(buf)
	defer func() {
		internal.RedeemBytesBuffer(buf)
		redeem()
	}()

	if err := d.encode(jw); err != nil {
		return fmt.Errorf("cannot marshal JSON: %w", err).Error()
	}

	return buf.String()
}

func (d *Document) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = d.store
	context.DO = d.DecodeOptions
	d.root.Decode(context)
	light.RedeemParentContext(context)

	return lex.Err()
}

func (d Document) encode(jw writers.Writer) error {
	context := light.BorrowParentContext()
	context.W = jw
	context.S = d.store
	context.EO = d.EncodeOptions

	d.root.Encode(context)
	light.RedeemParentContext(context)

	return jw.Err()
}
