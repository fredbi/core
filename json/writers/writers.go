package writers

import (
	"io"

	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/json/types"
)

type BaseWriter interface {
	// write delimiters
	StartObject()
	EndObject()
	StartArray()
	EndArray()
	Comma()
	Colon()

	// write native go types
	Null()
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

type TokenWriter interface {
	Token(token.T) // write a token

	BaseWriter
}

// VerbatimWriter is the interface for types that know how to write verbatim JSON tokens and values.
type VerbatimWriter interface {
	VerbatimToken(token.VT)             // write a verbatim token
	VerbatimValue(values.VerbatimValue) // write a verbatim value

	BaseWriter
}

type Flusher interface {
	Flush() error
}

type StoreWriter interface {
	// write data from a [stores.Store]
	Key(values.InternedKey)
	Value(values.Value)

	BaseWriter
}

type JSONWriter interface {
	// write JSON types
	JSONString(types.String)
	JSONNumber(types.Number)
	JSONBoolean(types.Boolean)
	JSONNull(types.NullType)

	BaseWriter
}
