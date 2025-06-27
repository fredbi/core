package writers

import (
	"io"

	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/types"
)

// Writer is the interface for types that know how to write JSON tokens and values.
type Writer interface {
	Token(token.T) // write a token

	DataWriter
}

// Writer is the interface for types that know how to write verbatim JSON tokens and values.
type VerbatimWriter interface {
	VerbatimToken(token.VT)             // write a verbatim token
	VerbatimValue(stores.VerbatimValue) // write a verbatim value

	DataWriter
}

type Flusher interface {
	Flush() error
}

// DataWriter is the common interface for [Writer] and [VerbatimWriter].
//
// It knows how to write JSON data from a [stores.Store], JSON types as well as go values.
type DataWriter interface {
	// write data from a [stores.Store]
	Key(stores.InternedKey)
	Value(stores.Value)
	Null()

	// write delimiters
	StartObject()
	EndObject()
	StartArray()
	EndArray()
	Comma()
	Colon()

	// write JSON types
	JSONString(types.String)
	JSONNumber(types.Number)
	JSONBoolean(types.Boolean)
	JSONNull(types.NullType)

	// write native go types
	String(string)
	StringBytes([]byte)
	StringRunes([]rune)
	StringCopy(io.Reader)

	Raw([]byte)
	RawCopy(io.Reader)

	Number(any)
	NumberBytes([]byte)
	NumberCopy(io.Reader)

	Bool(bool)

	// Size yields the number of bytes written so far
	Size() int64

	types.ErrStateSetter
	types.Resettable
	types.WithErrState
}
