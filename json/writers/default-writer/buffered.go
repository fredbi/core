package writer

import (
	"encoding"
	"errors"
	"fmt"
	"io"
	"math/big"
	"runtime"

	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/json/types"
	"github.com/fredbi/core/json/writers"
	"github.com/fredbi/core/swag/conv"
)

var (
	_ writers.StoreWriter    = &Buffered{}
	_ writers.JSONWriter     = &Buffered{}
	_ writers.TokenWriter    = &Buffered{}
	_ writers.VerbatimWriter = &Buffered{}
)

// Buffered JSON writer.
// [Buffered] implements
// [writers.StoreWriter], [writers.JSONWriter], [writers.TokenWriter] and [writers.VerbatimWriter].
//
// It knows how to render JSON tokens and JSON values to an underlying [io.Writer].
//
// When the caller is done writing values, [Buffered.Flush] must be called to ensure
// that all writes are passed to the [io.Writer].
//
// Strings are escaped with default JSON escaping rule for tabs, new lines, line feeds, backslashes and double quotes.
//
// There is no attempt to do anything special regarding empty or null values:
//
//   - an undefined value (or nil data) is not rendered
//   - a null value is necessarily defined explicitly and is rendered with the "null" token
//
// # Concurrency
//
// [Buffered] is not intended for concurrent use.
type Buffered struct {
	baseWriter
	*bufferedOptions
}

func NewBuffered(w io.Writer, opts ...BufferedOption) *Buffered {
	writer := &Buffered{
		baseWriter: baseWriter{
			w: w,
		},
		bufferedOptions: bufferedOptionsWithDefaults(opts), // always borrow options from the pool
	}

	// when using New, borrowed inner resources must be relinquished when the gc claims the writer.
	runtime.AddCleanup(writer, func(o *bufferedOptions) {
		if o != nil {
			o.redeem()
			poolOfBufferedOptions.Redeem(o)
		}
	}, writer.bufferedOptions)

	return writer
}

func (w *Buffered) Reset() {
	w.baseWriter.Reset()
	if w.bufferedOptions != nil {
		w.bufferedOptions.Reset()
	}
}

// Flush the internal buffer of the [Buffered] writer to the underlying [io.Writer].
func (w *Buffered) Flush() error {
	if w.err != nil {
		return w.err
	}

	w.flush()

	return w.err
}

// Comma writes a comma separator, ','.
func (w *Buffered) Comma() {
	w.writeSingleByte(comma)
}

// Colon writes a colon separator, ':'.
func (w *Buffered) Colon() {
	w.writeSingleByte(colon)
}

// EndArray writes an end-of-array separator, i.e. ']'.
func (w *Buffered) EndArray() {
	w.writeSingleByte(closingSquareBracket)
}

// EndObject writes an end-of-object separator, i.e. '}'.
func (w *Buffered) EndObject() {
	w.writeSingleByte(closingBracket)
}

// StartArray writes a start-of-array separator, i.e. '['.
func (w *Buffered) StartArray() {
	w.writeSingleByte(openingSquareBracket)
}

// StartObject writes a start-of-object separator, i.e. '{'.
func (w *Buffered) StartObject() {
	w.writeSingleByte(openingBracket)
}

// Bool writes a boolean value as JSON.
func (w *Buffered) Bool(v bool) {
	if w.err != nil {
		return
	}

	if v {
		w.writeBinary(trueBytes)

		return
	}

	w.writeBinary(falseBytes)
}

// Raw appends raw bytes to the buffer, without quotes and without escaping.
func (w *Buffered) Raw(data []byte) {
	if w.err != nil || len(data) == 0 {
		return
	}

	w.writeBinary(data)
}

// String writes a string as a JSON string value enclosed by double quotes, with escaping.
//
// The empty string is a legit input.
func (w *Buffered) String(s string) {
	if w.err != nil {
		return
	}

	w.writeTextString(s)
}

// StringBytes writes a slice of bytes as a JSON string enclosed by double quotes ('"'), with escaping.
//
// An empty slice is a legit input.
func (w *Buffered) StringBytes(data []byte) {
	if w.err != nil || data == nil {
		return
	}

	w.writeText(data)
}

// StringRunes writes a slice of bytes as a JSON string enclosed by double quotes ('"'), with escaping.
//
// An empty slice is a legit input.
func (w *Buffered) StringRunes(data []rune) {
	writeTextRunes(w, data)
}

// NumberBytes writes a slice of bytes as a JSON number.
//
// No check is carried out.
func (w *Buffered) NumberBytes(data []byte) {
	if w.err != nil || len(data) == 0 {
		return
	}

	w.writeBinary(data)
}

// StringCopy writes the bytes consumed from an [io.Reader] as a JSON string enclosed by double quotes ('"'), with escaping.
func (w *Buffered) StringCopy(r io.Reader) {
	stringCopy(w, r)
}

// NumberCopy writes the bytes consumed from an [io.Reader] as a JSON number.
//
// No check is carried out.
func (w *Buffered) NumberCopy(r io.Reader) {
	w.RawCopy(r)
}

// RawCopy writes the bytes consumed from an [io.Reader], without quotes and without escaping.
func (w *Buffered) RawCopy(r io.Reader) {
	if w.err != nil {
		return
	}

	bufHolder, redeemReadBuffer := poolOfReadBuffers.BorrowWithRedeem()
	buf := bufHolder.Slice()

	for {
		n, err := r.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			w.err = err

			break
		}

		if n > 0 {
			w.writeBinary(buf[:n])
			if w.err != nil {
				break
			}
		}

		if n == 0 || (err != nil && errors.Is(err, io.EOF)) {
			break
		}
	}

	redeemReadBuffer()
}

// JSONString writes a JSON value of [types.String].
//
// Nothing is written if the value is undefined.
func (w *Buffered) JSONString(value types.String) {
	if w.err != nil || !value.IsDefined() || len(value.Value) == 0 {
		return
	}

	w.writeText(value.Value)
}

// JSONNumber writes a JSON value of [types.Number].
//
// Nothing is written if the value is undefined.
func (w *Buffered) JSONNumber(value types.Number) {
	if w.err != nil || !value.IsDefined() || len(value.Value) == 0 {
		return
	}

	w.writeBinary(value.Value)
}

// JSONBoolean writes a JSON value of [types.Boolean].
//
// Nothing is written if the value is undefined.
func (w *Buffered) JSONBoolean(value types.Boolean) {
	if w.err != nil || !value.IsDefined() {
		return
	}

	w.Bool(value.Value)
}

// JSONNull writes a JSON value of [types.NullType], i.e. the "null" token.
//
// Nothing is written if the value is undefined.
func (w *Buffered) JSONNull(value types.NullType) {
	if w.err != nil || !value.IsDefined() {
		return
	}

	w.writeBinary(nullToken)
}

// Value writes a [values.Value]
func (w *Buffered) Value(v values.Value) {
	switch v.Kind() {
	case token.String:
		w.StringBytes(v.StringValue().Value)
	case token.Number:
		w.NumberBytes(v.NumberValue().Value)
	case token.Boolean:
		w.Bool(v.Bool())
	case token.Null:
		w.Null()
	default:
		// skip
	}
}

// Null writes a null token ("null").
func (w *Buffered) Null() {
	if w.err != nil {
		return
	}

	w.writeBinary(nullToken)
}

// Key write a key [values.InternedKey] followed by a colon (":").
func (w *Buffered) Key(key values.InternedKey) {
	w.String(key.String())
	w.Colon()
}

// Number writes a number from any native numerical go type, except complex numbers.
//
// Types from the math/big package are supported: [big.Int], [big.Rat], [big.Float].
//
// Numbers provided as a slice of bytes are also supported (no check is carried out).
//
// The method panics if the argument is not a numerical type or []byte.
func (w *Buffered) Number(v any) {
	if w.err != nil {
		return
	}

	holder, redeem := poolOfNumberBuffers.BorrowWithRedeem()
	defer redeem()
	dst := holder.Slice()

	switch n := v.(type) {
	case uint8:
		w.writeBinary(conv.AppendUinteger(dst, n))
	case uint16:
		w.writeBinary(conv.AppendUinteger(dst, n))
	case uint32:
		w.writeBinary(conv.AppendUinteger(dst, n))
	case uint64:
		w.writeBinary(conv.AppendUinteger(dst, n))
	case uint:
		w.writeBinary(conv.AppendUinteger(dst, n))
	case int8:
		w.writeBinary(conv.AppendInteger(dst, n))
	case int16:
		w.writeBinary(conv.AppendInteger(dst, n))
	case int32:
		w.writeBinary(conv.AppendInteger(dst, n))
	case int64:
		w.writeBinary(conv.AppendInteger(dst, n))
	case int:
		w.writeBinary(conv.AppendInteger(dst, n))
	case float32:
		w.writeBinary(conv.AppendFloat(dst, n))
	case float64:
		w.writeBinary(conv.AppendFloat(dst, n))
	case []byte:
		w.writeBinary(n)
	case *big.Int:
		if n == nil {
			return
		}
		w.append(n)
		return
	case big.Int:
		w.append(&n)
		return
	case *big.Rat:
		if n == nil {
			return
		}
		f, _ := n.Float64()
		w.writeBinary(conv.AppendFloat(dst, f))
	case big.Rat:
		f, _ := n.Float64()
		w.writeBinary(conv.AppendFloat(dst, f))
	case *big.Float:
		if n == nil {
			return
		}
		w.appendFloat(n)
		return
	case big.Float:
		w.appendFloat(&n)
		return
	default:
		panic(fmt.Errorf(
			"expected argument to Number() to be of a numerical type, but got: %T: %w",
			v, ErrDefaultWriter,
		))
	}
}

// Token writes a token [token.T] from a lexer.
//
// For key tokens, you'd need to call explicitly with the following colon token.
func (w *Buffered) Token(tok token.T) {
	if w.err != nil {
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
	case token.String, token.Key:
		w.writeText(tok.Value())
	case token.Number:
		w.NumberBytes(tok.Value())
	case token.Boolean:
		w.Bool(tok.Bool())
	case token.Null:
		w.Null()
	case token.EOF:
		fallthrough
	default:
		// ignore
	}
}

// VerbatimToken writes a verbatim token [token.VT] from a verbatim lexer.
//
// Non-significant white-space preceding each token is written to the buffer.
func (w *Buffered) VerbatimToken(tok token.VT) {
	if w.err != nil {
		return
	}

	w.writeBinary(tok.Blanks())
	w.Token(tok.T)
}

func (w *Buffered) VerbatimValue(value values.VerbatimValue) {
	if w.err != nil {
		return
	}

	w.writeBinary(value.Blanks())
	w.Value(value.Value)
}

func (w *Buffered) flush() {
	n, err := w.w.Write(w.buffer)
	w.inc(n)
	w.err = err
	w.buffer = w.buffer[:0]
}

// redeem inner resources
func (w *Buffered) redeem() {
	if w.bufferedOptions != nil {
		w.bufferedOptions.redeem() // redeem the inner buffer
		poolOfBufferedOptions.Redeem(w.bufferedOptions)
		w.bufferedOptions = nil
	}
}

func (w *Buffered) append(n encoding.TextAppender) {
	appendRaw(w, n)
}

func (w *Buffered) appendFloat(n *big.Float) {
	appendFloat(w, n)
}

func (w *Buffered) writeSingleByte(c byte) {
	if len(w.buffer) == cap(w.buffer) {
		w.flush()
	}

	if w.err != nil {
		return
	}

	w.buffer = append(w.buffer, c)
}

func (w *Buffered) writeBinary(data []byte) {
	var offset int

	for offset < len(data) {
		if len(w.buffer) == cap(w.buffer) {
			w.flush()
			if w.err != nil {
				return
			}
		}

		chunkSize := min(len(data[offset:]), cap(w.buffer)-len(w.buffer))
		w.buffer = append(w.buffer, data[offset:offset+chunkSize]...)

		offset += chunkSize
	}
}

func (w *Buffered) writeTextString(data string) {
	writeTextString(w, data)
}

func (w *Buffered) writeText(data []byte) {
	writeText(w, data)
}
