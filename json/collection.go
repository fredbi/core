package json

import (
	"encoding"
	"encoding/json"
	"io"
	"iter"

	"github.com/fredbi/core/json/stores"
)

var (
	_ json.Marshaler        = Collection{}
	_ encoding.TextAppender = Collection{}
)

// Collection is a collection of [Document] s, that share the same options.
//
// It can serve as a factory to produce [Document] s using [Collection.DecodeAppend].
//
// It marshals as an array of [Document]s.
type Collection struct {
	options

	documents []document
}

// NewCollection builds a new empty [Collection] of [Document] s.
func NewCollection(opts ...Option) *Collection {
	return &Collection{
		options: optionsWithDefaults(opts),
	}
}

// Store returns the underlying [stores.Store] where JSON values are kept.
func (c Collection) Store() stores.Store {
	return c.store
}

// Append a [Document] to the [Collection].
//
// If the [Collection] is empty, the options will be taken from the first [Document] appended.
//
// # Limitations
//
//   - [Document] s that are based on a different [stores.Store] won't be added and will be skipped.
func (c *Collection) Append(docs ...Document) {
	if len(docs) == 0 {
		return
	}

	if len(c.documents) == 0 {
		c.options = docs[0].options
	}

	for _, doc := range docs {
		if doc.Store() != c.Store() {
			// skip incompatible documents
			continue
		}

		c.documents = append(c.documents, doc.document)
	}
}

// DecodeAppend decodes a [Document] from the provided reader and appends it to the [Collection].
func (c *Collection) DecodeAppend(r io.Reader) error {
	lex, redeem := c.lexerFromReaderFactory(r)
	defer redeem()

	doc := Document{
		options: c.options,
	}
	if err := doc.decode(lex); err != nil {
		return err
	}

	c.documents = append(c.documents, doc.document)

	return nil
}

// Documents iterates over the [Document] s in the [Collection].
func (c *Collection) Documents() iter.Seq[Document] {
	return func(yield func(Document) bool) {
		for _, doc := range c.documents {
			completeDoc := Document{
				options:  c.options,
				document: doc,
			}
			if !yield(completeDoc) {
				return
			}
		}
	}
}

// Document yields the [Document] at the index position in the [Collection].
//
// It panics if index >= [Collection.Len].
func (c *Collection) Document(index int) Document {
	doc := c.documents[index]

	return Document{
		options:  c.options,
		document: doc,
	}
}

// Encode a collection of [Document] s as a stream of JSON bytes.
func (c Collection) Encode(w io.Writer) error {
	jw, redeem := c.writerToWriterFactory(w)
	defer redeem()

	jw.StartArray()

	if len(c.documents) == 0 {
		jw.EndArray()

		return nil
	}

	doc := Document{
		options:  c.options,
		document: c.documents[0],
	}
	if err := doc.encode(jw); err != nil {
		return err
	}

	for _, d := range c.documents[1:] {
		jw.Comma()
		doc.document = d

		if err := doc.encode(jw); err != nil {
			return err
		}
	}

	jw.EndArray()

	return nil
}

// AppendText appends the JSON bytes to the provided buffer and returns the resulting slice.
func (c Collection) AppendText(b []byte) ([]byte, error) {
	w := poolOfAppendWriters.Borrow()
	w.b = b
	jw, redeem := c.writerToWriterFactory(w)
	defer func() {
		poolOfAppendWriters.Redeem(w)
		redeem()
	}()

	jw.StartArray()

	if len(c.documents) == 0 {
		jw.EndArray()

		return w.b, nil
	}

	doc := Document{
		options:  c.options,
		document: c.documents[0],
	}
	if err := doc.encode(jw); err != nil {
		return nil, err
	}

	for _, d := range c.documents[1:] {
		jw.Comma()
		doc.document = d

		if err := doc.encode(jw); err != nil {
			return nil, err
		}
	}

	jw.EndArray()

	return w.b, nil
}

// MarshalJSON marshals the [Collection] as an array of JSON documents.
func (c Collection) MarshalJSON() ([]byte, error) {
	buf := poolOfBuffers.Borrow()
	jw, redeem := c.writerToWriterFactory(buf)
	defer func() {
		poolOfBuffers.Redeem(buf)
		redeem()
	}()

	jw.StartArray()

	if len(c.documents) == 0 {
		jw.EndArray()

		return buf.Bytes(), nil
	}

	doc := Document{
		options:  c.options,
		document: c.documents[0],
	}
	if err := doc.encode(jw); err != nil {
		return nil, err
	}

	for _, d := range c.documents[1:] {
		jw.Comma()
		doc.document = d

		if err := doc.encode(jw); err != nil {
			return nil, err
		}
	}

	jw.EndArray()

	return buf.Bytes(), nil
}

// Len returns the number of [Document] s in the collection.
func (c Collection) Len() int {
	return len(c.documents)
}

// Reset the collection, so it may be recycled.
func (c *Collection) Reset() {
	c.documents = c.documents[:0]
}
