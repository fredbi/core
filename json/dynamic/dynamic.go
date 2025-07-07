package dynamic

import (
	"errors"
	"io"
	"iter"
	"slices"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/internal"
	"github.com/fredbi/core/json/lexers"
	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/writers"
	"github.com/fredbi/core/swag/conv"
)

// [JSON] holds the dynamic go data structure created
// when unmarshaling JSON into an untyped `interface{}` value.
//
// The inner structure is built from a JSON string using map[string]any for objects,
// []any for arrays, string for strings and float64 for numbers.
type JSON struct {
	options
	inner any
}

// Make a new [JSON] object.
func Make(opts ...Option) JSON {
	var inner any

	return JSON{
		options: optionsWithDefaults(opts),
		inner:   &inner,
	}
}

// TODO: data navigation methods (iterators) like for Document.

// ToJSON converts a [json.Document] into a dynamic [JSON] data structure.
func ToJSON(d json.Document, opts ...Option) JSON {
	//	j := Make(opts...)

	return JSON{} // TODO
}

// ToDocument builds a [json.Document] from a dynamic [JSON] data structure,
//
// i.e. akin to what you get when the standard library unmarshals into a "any" type.
// TODO
// Options: specify Store
func ToDocument(value JSON, opts ...json.Option) (json.Document, error) {
	// TODO b := json.NewBuilder()

	return json.EmptyDocument, errors.New("not imlemented")
}

// Interface returns the inner untyped go structure (type "any").
func (d JSON) Interface() any {
	return d.inner
}

func (d *JSON) Reset() {
	var inner any
	d.inner = &inner
}

func (d *JSON) Decode(r io.Reader) error {
	lex, redeem := d.lexerFromReaderFactory(r)
	defer redeem()

	return d.decodeInner(lex)
}

func (d *JSON) UnmarshalJSON(data []byte) error {
	lex, redeem := d.lexerFactory(data)
	defer redeem()

	return d.decodeInner(lex)
}

func (d *JSON) decodeInner(lex lexers.Lexer) error {
	d.inner = d.decode(lex)

	return lex.Err()
}

func (d JSON) Encode(w io.Writer) error {
	jw, redeem := d.writerToWriterFactory(w)
	defer redeem()

	return d.encodeInner(jw)
}

func (d JSON) MarshalJSON() ([]byte, error) {
	buf := internal.BorrowBytesBuffer()
	jw, redeem := d.writerToWriterFactory(buf)
	defer func() {
		internal.RedeemBytesBuffer(buf)
		redeem()
	}()

	err := d.encodeInner(jw)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (d JSON) AppendText(b []byte) ([]byte, error) {
	w := internal.BorrowAppendWriter()
	w.Set(b)
	jw, redeem := d.writerToWriterFactory(w)
	defer func() {
		internal.RedeemAppendWriter(w)
		redeem()
	}()

	err := d.encodeInner(jw)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

func (d JSON) encodeInner(jw writers.JSONWriter) error {
	d.encode(jw)

	return jw.Err()
}

func (d *JSON) decode(l lexers.Lexer) any {
	if !l.Ok() {
		return nil
	}

	for {
		tok := l.NextToken()
		if !l.Ok() {
			return nil
		}

		switch {
		case tok.IsStartObject():
			node := make(map[string]any)
			for key, value := range d.decodeObject(l) {
				if !l.Ok() {
					return nil
				}
				node[key] = value
			}

			return node

		case tok.IsStartArray():
			node := make([]any, 0, 10)
			for elem := range d.decodeArray(l) {
				node = append(node, elem)
				if !l.Ok() {
					return nil
				}
			}
			slices.Clip(node)

			return node

		case tok.IsNull():
			return nil
		case tok.IsBool():
			return tok.Bool()
		case tok.IsScalar():
			switch tok.Kind() {
			case token.String:
				return string(tok.Value())
			case token.Number:
				// TODO: smarter number conversion
				f, err := conv.ConvertFloat64(string(tok.Value()))
				if err != nil {
					l.SetErr(err)
					return nil
				}
				return f
			default:
				l.SetErr(codes.ErrInvalidToken)
				return nil
			}
		case tok.IsEOF():
			return nil
		default:
			// wrong
			l.SetErr(codes.ErrInvalidToken)
			return nil
		}
	}
}

func (d *JSON) decodeObject(l lexers.Lexer) iter.Seq2[string, any] {
	if !l.Ok() {
		return nil
	}

	return func(yield func(string, any) bool) {
		for {
			tok := l.NextToken()
			if !l.Ok() {
				return
			}

			if tok.IsEndObject() {
				// empty object
				return
			}

			if tok.IsKey() {
				l.SetErr(codes.ErrMissingKey)
				return
			}

			tok = l.NextToken() // skip the colon separator following the key
			if !tok.IsColon() {
				l.SetErr(codes.ErrKeyColon)
				return
			}

			key := string(tok.Value())

			v := Make()
			v.decode(l)
			if !l.Ok() {
				return
			}
			value := v.Interface()

			if !yield(key, value) {
				return
			}

			separator := l.NextToken()
			if !l.Ok() {
				return
			}

			if separator.IsComma() {
				continue
			}

			if separator.IsEndObject() {
				return
			}

			l.SetErr(codes.ErrInvalidToken)

			return
		}
	}
}

func (d *JSON) decodeArray(l lexers.Lexer) iter.Seq[any] {
	if !l.Ok() {
		return nil
	}

	return func(yield func(any) bool) {
		for {
			tok := l.NextToken()
			if !l.Ok() {
				return
			}

			if tok.IsEndArray() {
				// empty object
				return
			}

			v := Make()
			v.decode(l)
			if !l.Ok() {
				return
			}
			elem := v.Interface()

			if !yield(elem) {
				return
			}

			separator := l.NextToken()
			if !l.Ok() {
				return
			}

			if separator.IsComma() {
				continue
			}

			if separator.IsEndArray() {
				return
			}

			l.SetErr(codes.ErrMissingComma)
			return
		}
	}
}
