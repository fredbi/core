package writer

import (
	"io"
	"runtime"

	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/json/types"
	"github.com/fredbi/core/json/writers"
)

var (
	_ writers.StoreWriter = &Indented{}
	_ writers.JSONWriter  = &Indented{}
	_ writers.TokenWriter = &Indented{}
)

type Indented struct {
	*Buffered2
	*indentedOptions
	level           int
	redeemBuffered2 *Buffered2 // mark that the Buffered2 must be redeemed
	lastSeparator   byte
	containerOnHold bool
}

func NewIndented(w io.Writer, opts ...IndentedOption) *Indented {
	o := indentedOptionsWithDefaults(opts)
	writer := &Indented{
		Buffered2:       NewBuffered2(w, o.applyBufferedOptions...),
		indentedOptions: o,
	}

	// when using New, borrowed inner resources must be relinquished when the gc claims the writer.
	runtime.AddCleanup(writer, func(o *indentedOptions) {
		if o != nil {
			o.redeem()

			poolOfIndentedOptions.Redeem(o)
		}
	}, writer.indentedOptions)

	return writer
}

func (w *Indented) Reset() {
	w.lastSeparator = 0
	w.level = 0

	if w.Buffered2 != nil {
		w.Buffered2.Reset()
	}

	if w.indentedOptions != nil {
		w.indentedOptions.Reset()
	}
}

func (w *Indented) Flush() error {
	w.releaseHold(true)

	return w.Buffered2.Flush()
}

// Comma writes a comma separator, ','.
func (w *Indented) Comma() {
	w.releaseHold(true)
	w.Buffered2.Comma()
	w.writeNewlineIndent()
	w.lastSeparator = comma
}

// Colon writes a colon separator, ':'.
func (w *Indented) Colon() {
	w.releaseHold(true)
	w.Buffered2.Colon()
	w.jw.writeSingleByte(space)
	w.lastSeparator = colon
}

// EndArray writes an end-of-array separator, i.e. ']'.
func (w *Indented) EndArray() {
	// if the previous written was [, do not indent on empty array
	if w.containerOnHold {
		w.releaseHold(w.lastSeparator != openingSquareBracket)

		w.Buffered2.EndArray()
		w.lastSeparator = closingSquareBracket
		w.level--

		return
	}

	// no hold: close array and indent normally
	w.level--
	w.writeNewlineIndent()
	w.Buffered2.EndArray()
	w.lastSeparator = closingSquareBracket
}

// EndObject writes an end-of-object separator, i.e. '}'.
func (w *Indented) EndObject() {
	// if the previous written was {, do not indent on empty object
	if w.containerOnHold {
		w.releaseHold(w.lastSeparator != openingBracket)

		w.Buffered2.EndObject()
		w.lastSeparator = closingBracket
		w.level--

		return
	}

	// no hold: close object and indent normally
	w.level--
	w.writeNewlineIndent()
	w.Buffered2.EndObject()
	w.lastSeparator = closingBracket
}

// StartArray writes a start-of-array separator, i.e. '['.
func (w *Indented) StartArray() {
	// if it was previously on hold, and we got a new token, release previous container
	w.releaseHold(true)

	// this is a new array: keep containerOnHold

	// put on hold until next write or flush
	w.containerOnHold = true
	w.lastSeparator = openingSquareBracket
}

// StartObject writes a start-of-object separator, i.e. '{'.
func (w *Indented) StartObject() {
	// if it was previously on hold, and we got a new token, release previous container
	w.releaseHold(true)

	// this is a new object: keep containerOnHold

	// put on hold until next write or flush
	w.containerOnHold = true
	w.lastSeparator = openingBracket
}

func (w *Indented) Key(key values.InternedKey) {
	w.releaseHold(true)
	w.Buffered2.String(key.String())
	w.Colon()
}

func (w *Indented) Token(tok token.T) {
	if !w.Ok() {
		return
	}

	switch tok.Kind() {
	case token.Delimiter:
		switch tok.Delimiter() {
		case token.OpeningBracket:
			w.StartObject()
		case token.ClosingBracket:
			w.EndObject()
		case token.OpeningSquareBracket:
			w.StartArray()
		case token.ClosingSquareBracket:
			w.EndArray()
		case token.Comma:
			w.Comma()
		case token.Colon:
			w.Colon()
		default:
			// ignore
		}
	default:
		w.releaseHold(true)
		w.Buffered2.Token(tok)
	}
}
func (w *Indented) Bool(v bool) {
	w.releaseHold(true)
	w.Buffered2.Bool(v)
}
func (w *Indented) Raw(data []byte) {
	w.releaseHold(true)
	w.Buffered2.Raw(data)
}

func (w *Indented) String(s string) {
	w.releaseHold(true)
	w.Buffered2.String(s)
}

func (w *Indented) StringBytes(data []byte) {
	w.releaseHold(true)
	w.Buffered2.StringBytes(data)
}
func (w *Indented) StringRunes(data []rune) {
	w.releaseHold(true)
	w.Buffered2.StringRunes(data)
}
func (w *Indented) NumberBytes(data []byte) {
	w.releaseHold(true)
	w.Buffered2.NumberBytes(data)
}
func (w *Indented) NumberCopy(r io.Reader) {
	w.releaseHold(true)
	w.Buffered2.NumberCopy(r)
}
func (w *Indented) RawCopy(r io.Reader) {
	w.releaseHold(true)
	w.Buffered2.RawCopy(r)
}
func (w *Indented) StringCopy(r io.Reader) {
	w.releaseHold(true)
	w.Buffered2.StringCopy(r)
}
func (w *Indented) JSONString(value types.String) {
	w.releaseHold(true)
	w.Buffered2.JSONString(value)
}
func (w *Indented) JSONNumber(value types.Number) {
	w.releaseHold(true)
	w.Buffered2.JSONNumber(value)
}
func (w *Indented) JSONBoolean(value types.Boolean) {
	w.releaseHold(true)
	w.Buffered2.JSONBoolean(value)
}
func (w *Indented) JSONNull(value types.NullType) {
	w.releaseHold(true)
	w.Buffered2.JSONNull(value)
}
func (w *Indented) Value(v values.Value) {
	w.releaseHold(true)
	w.Buffered2.Value(v)
}
func (w *Indented) Null() {
	w.releaseHold(true)
	w.Buffered2.Null()
}
func (w *Indented) Number(v any) {
	w.releaseHold(true)
	w.Buffered2.Number(v)
}

func (w *Indented) redeem() {
	if w.redeemBuffered2 != nil { // this is hydrated when borrowing from a pool and remains nil when created with New
		RedeemBuffered2(w.redeemBuffered2)
	}

	if w.indentedOptions != nil {
		w.indentedOptions.redeem()

		poolOfIndentedOptions.Redeem(w.indentedOptions)
		w.indentedOptions = nil
	}
}

func (w *Indented) releaseHold(wantIndent bool) {
	if !w.containerOnHold {
		return
	}

	switch w.lastSeparator {
	case openingSquareBracket:
		w.Buffered2.StartArray()
	case openingBracket:
		w.Buffered2.StartObject()
	}

	w.level++
	if wantIndent {
		w.writeNewlineIndent()
	}
	w.containerOnHold = false
}

func (w *Indented) writeNewlineIndent() {
	if !w.Ok() {
		return
	}

	w.jw.writeSingleByte(newline)

	for range w.level {
		w.jw.writeBinary(w.indent)
	}
}
