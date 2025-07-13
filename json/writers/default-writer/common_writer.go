package writer

import (
	"encoding"
	"errors"
	"fmt"
	"io"
	"math/big"
	"unicode/utf8"

	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/json/types"
	"github.com/fredbi/core/swag/conv"
)

// commonWriter implements most methods for writers based on
// a few primitive methods defined by the [wrt] interface.
//
// Wrapping a commonWriter is roughly just as fast as repeating code in each writer (only about 5% slower).
type commonWriter[T wrt] struct {
	jw T
}

// Comma writes a comma separator, ','.
func (w *commonWriter[T]) Comma() {
	w.jw.writeSingleByte(comma)
}

// Colon writes a colon separator, ':'.
func (w *commonWriter[T]) Colon() {
	w.jw.writeSingleByte(colon)
}

// EndArray writes an end-of-array separator, i.e. ']'.
func (w *commonWriter[T]) EndArray() {
	w.jw.writeSingleByte(closingSquareBracket)
}

// EndObject writes an end-of-object separator, i.e. '}'.
func (w *commonWriter[T]) EndObject() {
	w.jw.writeSingleByte(closingBracket)
}

// StartArray writes a start-of-array separator, i.e. '['.
func (w *commonWriter[T]) StartArray() {
	w.jw.writeSingleByte(openingSquareBracket)
}

// StartObject writes a start-of-object separator, i.e. '{'.
func (w *commonWriter[T]) StartObject() {
	w.jw.writeSingleByte(openingBracket)
}

// Bool writes a boolean value as JSON.
func (w *commonWriter[T]) Bool(v bool) {
	if w.jw.Err() != nil {
		return
	}

	if v {
		w.jw.writeBinary(trueBytes)

		return
	}

	w.jw.writeBinary(falseBytes)
}

// Raw appends raw bytes to the buffer, without quotes and without escaping.
func (w *commonWriter[T]) Raw(data []byte) {
	if w.jw.Err() != nil || len(data) == 0 {
		return
	}

	w.jw.writeBinary(data)
}

// String writes a string as a JSON string value enclosed by double quotes, with escaping.
//
// The empty string is a legit input.
func (w *commonWriter[T]) String(s string) {
	if w.jw.Err() != nil {
		return
	}

	w.writeTextString(s)
}

// StringBytes writes a slice of bytes as a JSON string enclosed by double quotes ('"'), with escaping.
//
// An empty slice is a legit input.
func (w *commonWriter[T]) StringBytes(data []byte) {
	if w.jw.Err() != nil || data == nil {
		return
	}

	w.writeText(data)
}

// StringRunes writes a slice of bytes as a JSON string enclosed by double quotes ('"'), with escaping.
//
// An empty slice is a legit input.
func (w *commonWriter[T]) StringRunes(data []rune) {
	if w.jw.Err() != nil || data == nil {
		return
	}
	holder, redeem := poolOfEscapedBuffers.BorrowWithSizeAndRedeem(len(data) * utf8.MaxRune)
	defer redeem()

	buf := holder.Slice()
	for _, r := range data {
		buf = utf8.AppendRune(buf, r)
	}

	w.writeText(buf)
}

// NumberBytes writes a slice of bytes as a JSON number.
//
// No check is carried out.
func (w *commonWriter[T]) NumberBytes(data []byte) {
	if w.jw.Err() != nil || len(data) == 0 {
		return
	}

	w.jw.writeBinary(data)
}

// NumberCopy writes the bytes consumed from an [io.Reader] as a JSON number.
//
// No check is carried out.
func (w *commonWriter[T]) NumberCopy(r io.Reader) {
	w.RawCopy(r)
}

// RawCopy writes the bytes consumed from an [io.Reader], without quotes and without escaping.
func (w *commonWriter[T]) RawCopy(r io.Reader) {
	if w.jw.Err() != nil {
		return
	}

	bufHolder, redeemReadBuffer := poolOfReadBuffers.BorrowWithRedeem()
	buf := bufHolder.Slice()

	for {
		n, err := r.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			w.jw.SetErr(err)

			break
		}

		if n > 0 {
			w.jw.writeBinary(buf[:n])
			if w.jw.Err() != nil {
				break
			}
		}

		if n == 0 || (err != nil && errors.Is(err, io.EOF)) {
			break
		}
	}

	redeemReadBuffer()
}

func (w *commonWriter[T]) StringCopy(r io.Reader) {
	w.jw.writeSingleByte(quote)
	if w.jw.Err() != nil {
		return
	}

	var remainder []byte
	bufHolder, redeemReadBuffer := poolOfReadBuffers.BorrowWithRedeem()
	extraBufHolder, redeemExtraBuf := poolOfEscapedBuffers.BorrowWithSizeAndRedeem(bufHolder.Len() + utf8.UTFMax)
	escapedHolder, redeemEscaped := poolOfEscapedBuffers.BorrowWithSizeAndRedeem(bufHolder.Len())
	defer func() {
		redeemReadBuffer()
		redeemEscaped()
		redeemExtraBuf()
	}()

	buf := bufHolder.Slice()
	escapedBuffer := escapedHolder.Slice()
	extra := extraBufHolder.Slice()

	for {
		n, err := r.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			w.jw.SetErr(err)

			return
		}

		if n > 0 {
			if len(extra) > 0 {
				// if the previous read reported an incomplete rune, the incomplete part is prepended to the input now
				extra = append(extra, buf[:n]...)
				escapedBuffer, remainder = escapedBytes(extra, escapedBuffer)
				extra = extra[:0]
			} else {
				escapedBuffer, remainder = escapedBytes(buf[:n], escapedBuffer)
			}

			w.jw.writeBinary(escapedBuffer)
			if w.jw.Err() != nil {
				return
			}

			if len(remainder) > 0 {
				if len(remainder) >= utf8.UTFMax {
					w.jw.SetErr(fmt.Errorf("unexpected incomplete rune: %c: %w", remainder, ErrDefaultWriter))

					return
				}

				extra = extra[:0]
				extra = append(extra, remainder...)
			}
		}

		if n == 0 || (err != nil && errors.Is(err, io.EOF)) {
			if len(remainder) > 0 {
				w.jw.SetErr(fmt.Errorf("unexpected incomplete rune at end of input: %c: %w", remainder, ErrDefaultWriter))

				return
			}

			break
		}
	}

	w.jw.writeSingleByte(quote)
}

// JSONString writes a JSON value of [types.String].
//
// Nothing is written if the value is undefined.
func (w *commonWriter[T]) JSONString(value types.String) {
	if w.jw.Err() != nil || !value.IsDefined() || len(value.Value) == 0 {
		return
	}

	w.writeText(value.Value)
}

// JSONNumber writes a JSON value of [types.Number].
//
// Nothing is written if the value is undefined.
func (w *commonWriter[T]) JSONNumber(value types.Number) {
	if w.jw.Err() != nil || !value.IsDefined() || len(value.Value) == 0 {
		return
	}

	w.jw.writeBinary(value.Value)
}

// JSONBoolean writes a JSON value of [types.Boolean].
//
// Nothing is written if the value is undefined.
func (w *commonWriter[T]) JSONBoolean(value types.Boolean) {
	if w.jw.Err() != nil || !value.IsDefined() {
		return
	}

	w.Bool(value.Value)
}

// JSONNull writes a JSON value of [types.NullType], i.e. the "null" token.
//
// Nothing is written if the value is undefined.
func (w *commonWriter[T]) JSONNull(value types.NullType) {
	if w.jw.Err() != nil || !value.IsDefined() {
		return
	}

	w.jw.writeBinary(nullToken)
}

// Value writes a [values.Value]
func (w *commonWriter[T]) Value(v values.Value) {
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
func (w *commonWriter[T]) Null() {
	if w.jw.Err() != nil {
		return
	}

	w.jw.writeBinary(nullToken)
}

// Key write a key [values.InternedKey] followed by a colon (":").
func (w *commonWriter[T]) Key(key values.InternedKey) {
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
func (w *commonWriter[T]) Number(v any) {
	if w.jw.Err() != nil {
		return
	}

	holder, redeem := poolOfNumberBuffers.BorrowWithRedeem()
	defer redeem()
	dst := holder.Slice()

	switch n := v.(type) {
	case uint8:
		w.jw.writeBinary(conv.AppendUinteger(dst, n))
	case uint16:
		w.jw.writeBinary(conv.AppendUinteger(dst, n))
	case uint32:
		w.jw.writeBinary(conv.AppendUinteger(dst, n))
	case uint64:
		w.jw.writeBinary(conv.AppendUinteger(dst, n))
	case uint:
		w.jw.writeBinary(conv.AppendUinteger(dst, n))
	case int8:
		w.jw.writeBinary(conv.AppendInteger(dst, n))
	case int16:
		w.jw.writeBinary(conv.AppendInteger(dst, n))
	case int32:
		w.jw.writeBinary(conv.AppendInteger(dst, n))
	case int64:
		w.jw.writeBinary(conv.AppendInteger(dst, n))
	case int:
		w.jw.writeBinary(conv.AppendInteger(dst, n))
	case float32:
		w.jw.writeBinary(conv.AppendFloat(dst, n))
	case float64:
		w.jw.writeBinary(conv.AppendFloat(dst, n))
	case []byte:
		w.jw.writeBinary(n)
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
		w.jw.writeBinary(conv.AppendFloat(dst, f))
	case big.Rat:
		f, _ := n.Float64()
		w.jw.writeBinary(conv.AppendFloat(dst, f))
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
func (w *commonWriter[T]) Token(tok token.T) {
	if w.jw.Err() != nil {
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
func (w *commonWriter[T]) VerbatimToken(tok token.VT) {
	if w.jw.Err() != nil {
		return
	}

	w.jw.writeBinary(tok.Blanks())
	w.Token(tok.T)
}

func (w *commonWriter[T]) VerbatimValue(value values.VerbatimValue) {
	if w.jw.Err() != nil {
		return
	}

	w.jw.writeBinary(value.Blanks())
	w.Value(value.Value)
}

// append writes down the result of AppendText.
//
// This borrows a temporary buffer to decode the result of AppendText()
func (w *commonWriter[T]) append(n encoding.TextAppender) {
	buf, redeem := poolOfNumberBuffers.BorrowWithRedeem()
	defer redeem()
	b := buf.Slice()

	b, err := n.AppendText(b)
	if err != nil {
		w.jw.SetErr(err)

		return
	}

	w.jw.writeBinary(b)
}

func (w *commonWriter[T]) appendFloat(n *big.Float) {
	buf, redeem := poolOfNumberBuffers.BorrowWithRedeem()
	defer redeem()
	b := buf.Slice()

	b = n.Append(b, 'g', int(n.MinPrec()))
	w.jw.writeBinary(b)
}

func (w *commonWriter[T]) writeTextString(input string) {
	stringBuffer, redeem := poolOfEscapedBuffers.BorrowWithSizeAndRedeem(len(input))
	defer redeem()
	data := stringBuffer.Slice()
	data = append(data, input...)

	w.writeText(data)
}

func (w *commonWriter[T]) writeText(data []byte) {
	w.jw.writeSingleByte(quote)
	if w.jw.Err() != nil {
		return
	}

	var remainder []byte

	// TODO: we need something more elaborate here: managing this pool takes more time than escaping the string
	// possibility: manage a free list of such buffers, so we only have to borrow from the pool ocasionally
	escapedHolder, redeemEscaped := poolOfEscapedBuffers.BorrowWithSizeAndRedeem(len(data)) // TODO: more elaborate pool
	escapedBuffer := escapedHolder.Slice()
	escapedBuffer, remainder = escapedBytes(data, escapedBuffer)
	w.jw.writeBinary(escapedBuffer)
	redeemEscaped()
	if len(remainder) > 0 {
		w.jw.SetErr(fmt.Errorf("incomplete rune at end of input: %c: %w", remainder, ErrDefaultWriter))

		return
	}

	w.jw.writeSingleByte(quote)
}
