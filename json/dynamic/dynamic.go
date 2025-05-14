package dynamic

import (
	"errors"
	"fmt"
	"iter"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/lexers"
	lexer "github.com/fredbi/core/json/lexers/default-lexer"
	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/writers"
	writer "github.com/fredbi/core/json/writers/default-writer"
	"github.com/fredbi/core/swag/conv"
)

// [JSON] holds the dynamic go data structure created
// when unmarshaling JSON into an untyped `interface{}` value.
//
// The inner structure is built from a JSON string using map[string]any for objects,
// []any for arrays, string for strings and float64 for numbers.
type JSON struct {
	inner any
}

// Make a new [JSON] object.
//
// TODO: options: support other numeric types than float64, use OrderedMap instead of map
func Make() JSON {
	var inner any
	return JSON{
		inner: &inner,
	}
}

// ToJSON converts a document into a "dynamic JSON" data structure.
func ToJSON(d json.Document) JSON {
	//return d.root.ToJSON(d.store, d.EncodeOptions)
	return JSON{} // TODO
}

// FromJSON builds a [Document] from a dynamic [JSON] data structure,
// i.e. akin to what you get when the standard library unmarshals into a "any" type.
// TODO
// Options: specify Store
func FromJSON(value JSON) (json.Document, error) {
	//return d.root.FromJSON(d.store, value, d.DecodeOptions)
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

func (d *JSON) UnmarshalJSON(data []byte) error {
	l := lexer.BorrowLexerWithBytes(data)
	defer func() {
		lexer.RedeemLexer(l)
	}()

	d.inner = d.decode(l)

	return l.Err()
}

func (d JSON) MarshalJSON() ([]byte, error) {
	w := writer.BorrowWriter()
	defer func() {
		writer.RedeemWriter(w)
	}()

	d.encode(w)

	if !w.Ok() {
		return nil, w.Err()
	}

	return nil, nil // TODO: BuildBytes()
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
			node := make([]any, 1)
			for elem := range d.decodeArray(l) {
				node = append(node, elem)
				if !l.Ok() {
					return nil
				}
			}
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

func (d *JSON) encode(w writers.Writer) {
	if !w.Ok() {
		return
	}

	if d.inner == nil {
		w.Null()

		return
	}

	switch inner := d.inner.(type) {
	case map[string]any:
		w.StartObject()
		if !w.Ok() {
			return
		}

		l := len(inner)

		if l == 0 {
			w.EndObject()

			return
		}

		i := 0
		for key, value := range inner {
			w.String(key)

			v := JSON{inner: value}
			v.encode(w)
			if i < l {
				w.Comma()
			}
			i++
		}

		w.EndObject()

		return

	case []any:
		w.StartArray()
		if !w.Ok() {
			return
		}

		l := len(inner)
		if l == 0 {
			w.EndArray()

			return
		}

		v0 := JSON{inner: inner[0]}
		v0.encode(w)

		for _, elem := range inner[1:] {
			w.Comma()
			v := JSON{inner: elem}
			v.encode(w)
		}

		w.EndArray()

		return

	case string:
		w.String(inner)
	case bool:
		w.Bool(inner)
	case float64:
		w.Float64(inner)
	case int64:
		w.Int64(inner)
	case uint64:
		w.Uint64(inner)
	default:
		w.SetErr(fmt.Errorf("invalid dynamic JSON type: %T", d.inner))
		return
	}
}
