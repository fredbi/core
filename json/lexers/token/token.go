package token

import (
	"fmt"
	"slices"
)

type (
	// Kind of JSON token, i.e. either a delimiter, a string, a number, a boolean or null.
	//
	// EOF is considered a special token that marks the end of a JSON stream.
	//
	// Strings and numbers are not converted to go string and go numeric types respectively:
	// the original value is kept as a slice of bytes.
	Kind uint8

	// KindDelimiter represents a JSON delimiter (i.e. ":", ",", "{", "}", "[", "]").
	KindDelimiter uint8
)

// T represents a JSON token.
//
// Tokens are immutable, short-lived objects.
//
// [T] maintains strings and numbers as slices of bytes representing an UTF8 string.
//
// It doesn't keep track of non-significant blank space or new lines.
//
// Escaped unicode sequences are unescaped as UTF8 runes.
//
// Limitation: JSON data based on a non-UTF8 character set need to be converted beforehand.
type T struct {
	value          []byte        // value for strings and numbers
	valueDelimiter KindDelimiter // value for delimiters
	kind           Kind          // kind of token
	valueBool      bool          // value for boolean tokens
}

// VT represents a verbatim JSON token.
//
// Like [T], verbatim tokens are immutable, short-lived objects.
//
// Like [T], it maintains strings and numbers as slices of bytes representing an UTF8 string.
//
// Unlike [T], [VT] maintains non-significant blank space and new lines, as well as
// escaped unicode sequences. Blanks ahead of any token (including EOF) are stored in the token.
//
// Limitation: JSON data based on a non-UTF8 character set need to be converted beforehand.
type VT struct {
	blanks []byte
	T
}

// None is a preallocated placeholder for any invalid or unrecognized JSON token.
var None = T{ //nolint:gochecknoglobals
	kind: Unknown,
}

// VNone is a preallocated placeholder for any invalid or unrecognized verbatim JSON token.
var VNone = VT{ //nolint:gochecknoglobals
	T: None,
}

// EOFToken is a preallocated placeholder returned whenever the lexer has reached
// the end of the input stream.
var EOFToken = T{ //nolint:gochecknoglobals
	kind: EOF,
}

// NullToken is a preallocated placeholder for "null" tokens.
var NullToken = T{ //nolint:gochecknoglobals
	kind: Null,
}

// JSON tokens.
const (
	// Unknown token.
	//
	// This result is associated with an error in the lexer.
	Unknown Kind = iota

	// Delimiter token, i.e. ",", ":", "{", "}", "[", "]".
	Delimiter

	// String token.
	//
	// Decoded strings are unescaped by [T], and left unchanged by [VT].
	String

	// Key string token.
	//
	// Decoded strings are unescaped by [T], and left unchanged by [VT].
	Key

	// Number JSON token.
	Number

	// Boolean token.
	Boolean

	// Null value token.
	Null

	// EOF signals that the lexer has reached the end of the input stream.
	EOF
)

// Delimiters.
const (
	// NotADelimiter is the zero value, used when the token is not a delimiter.
	NotADelimiter KindDelimiter = iota

	// Comma is ","
	Comma

	// Colon is ":"
	Colon

	// OpeningBracket is "{"
	OpeningBracket

	// ClosingBracket is "}"
	ClosingBracket

	// OpeningSquareBracket is "["
	OpeningSquareBracket

	// ClosingSquareBracket is "]"
	ClosingSquareBracket
)

// String representation of a delimiter.
func (d KindDelimiter) String() string {
	const (
		// delimiters for the JSON grammar
		openingBracket       = '{'
		closingBracket       = '}'
		openingSquareBracket = '['
		closingSquareBracket = ']'
		comma                = ','
		colon                = ':'
	)

	switch d {
	case Comma:
		return string(comma)
	case Colon:
		return string(colon)
	case OpeningBracket:
		return string(openingBracket)
	case ClosingBracket:
		return string(closingBracket)
	case OpeningSquareBracket:
		return string(openingSquareBracket)
	case ClosingSquareBracket:
		return string(closingSquareBracket)
	case NotADelimiter:
		return "not a delimiter"
	default:
		panic(fmt.Sprintf("invalid delimiter kind: %d", d))
	}
}

// IsClosing returns true for closing delimiters such as "}" or "]"
func (d KindDelimiter) IsClosing() bool {
	switch d {
	case ClosingBracket, ClosingSquareBracket:
		return true
	default:
		return false
	}
}

// AcceptValue returns true when the delimiter may be followed by a value token.
//
// Examples:
// ": true", "[\"abc\"]", ",123", {"abc" are legit
// but not:
// "} true", "] 123"
//
// Notice that {123 or {true are accepted: more context is needed to reject such constructs.
func (d KindDelimiter) AcceptValue() bool {
	switch d {
	case OpeningBracket, OpeningSquareBracket, Comma, Colon:
		return true
	default:
		return false
	}
}

// String representation of a kind of token.
func (k Kind) String() string {
	switch k {
	case Unknown:
		return "unknown"
	case Delimiter:
		return "delimiter"
	case String:
		return "string"
	case Key:
		return "key"
	case Number:
		return "number"
	case Boolean:
		return "boolean"
	case Null:
		return "null"
	case EOF:
		return "EOF"
	default:
		panic(fmt.Sprintf("invalid token kind: %d", k))
	}
}

// Make a token [T].
func Make(kind Kind, value []byte, delimiter KindDelimiter, valueBool bool) T {
	return T{
		kind:           kind,
		value:          value,
		valueDelimiter: delimiter,
		valueBool:      valueBool,
	}
}

// MakeVerbatim builds a verbatim token [VT].
func MakeVerbatim(
	kind Kind,
	value []byte,
	delimiter KindDelimiter,
	valueBool bool,
	blanks []byte,
) VT {
	return VT{
		T:      Make(kind, value, delimiter, valueBool),
		blanks: blanks,
	}
}

// MakeDelimiter builds a delimiter token [T].
func MakeDelimiter(delimiter KindDelimiter) T {
	return T{
		kind:           Delimiter,
		valueDelimiter: delimiter,
	}
}

// MakeVerbatim builds a verbatim delimiter token [VT].
func MakeVerbatimDelimiter(delimiter KindDelimiter, blanks []byte) VT {
	return VT{
		T:      MakeDelimiter(delimiter),
		blanks: blanks,
	}
}

// MakeWithValue builds a scalar string or number token [T].
func MakeWithValue(kind Kind, value []byte) T {
	return T{
		kind:  kind,
		value: value,
	}
}

// MakeVerbatimWithValue builds a verbatim scalar string or number token [VT].
func MakeVerbatimWithValue(kind Kind, value, blanks []byte) VT {
	return VT{
		T:      MakeWithValue(kind, value),
		blanks: blanks,
	}
}

// MakeBoolean builds a scalar boolean token [T].
func MakeBoolean(value bool) T {
	return T{
		kind:      Boolean,
		valueBool: value,
	}
}

// MakeVerbatimBoolean builds a scalar boolean token [VT].
func MakeVerbatimBoolean(value bool, blanks []byte) VT {
	return VT{
		T:      MakeBoolean(value),
		blanks: blanks,
	}
}

// MakeVerbatimNull builds a verbatim null token [VT].
func MakeVerbatimNull(blanks []byte) VT {
	return VT{
		T:      NullToken,
		blanks: blanks,
	}
}

func MakeVerbatimEOF(blanks []byte) VT {
	return VT{
		T:      EOFToken,
		blanks: blanks,
	}
}

// Value for String, Key and Number tokens.
func (t T) Value() []byte {
	return t.value
}

// Delimiter for delimiter tokens.
//
// The value is [NotADelimiter] for non-delimiter tokens.
func (t T) Delimiter() KindDelimiter {
	return t.valueDelimiter
}

// Blanks returns the leading blanks appearing before a token.
func (t VT) Blanks() []byte {
	return t.blanks
}

// Kind of token.
func (t T) Kind() Kind {
	return t.kind
}

// Bool value for boolean tokens.
func (t T) Bool() bool {
	return t.valueBool
}

// IsKnown checks if the token is valid.
func (t T) IsKnown() bool {
	return t.kind != Unknown
}

// IsScalar indicates is the Token represents a scalar value (null is not considered a scalar).
func (t T) IsScalar() bool {
	return t.kind == String || t.kind == Number || t.kind == Boolean
}

// IsNull indicates if the Token represents the null value.
func (t T) IsNull() bool {
	return t.kind == Null
}

// IsNull indicates if the Token represents a boolean value.
func (t T) IsBool() bool {
	return t.kind == Boolean
}

func (t T) IsStartObject() bool {
	return t.kind == Delimiter && t.valueDelimiter == OpeningBracket
}

func (t T) IsEndObject() bool {
	return t.kind == Delimiter && t.valueDelimiter == ClosingBracket
}

func (t T) IsStartArray() bool {
	return t.kind == Delimiter && t.valueDelimiter == OpeningSquareBracket
}

func (t T) IsEndArray() bool {
	return t.kind == Delimiter && t.valueDelimiter == ClosingSquareBracket
}

func (t T) IsComma() bool {
	return t.kind == Delimiter && t.valueDelimiter == Comma
}

func (t T) IsColon() bool {
	return t.kind == Delimiter && t.valueDelimiter == Colon
}

func (t T) IsKey() bool {
	return t.kind == Key
}

func (t T) IsEOF() bool {
	return t.kind == EOF
}

func (t T) IsDelimiter() bool {
	return t.kind == Delimiter
}

// Clone deep-clones a token.
//
// Memory to hold the token's string or numeric value will be
// freshly allocated.
func (t T) Clone() T {
	return T{
		value:          slices.Clone(t.value),
		valueDelimiter: t.valueDelimiter,
		kind:           t.kind,
		valueBool:      t.valueBool,
	}
}

// Clone deep-clones a verbatim token.
func (t VT) Clone() VT {
	return VT{
		T:      t.T.Clone(),
		blanks: slices.Clone(t.blanks),
	}
}

// String representation of a token.
//
// This is intended for logging or debug mostly.
func (t T) String() string {
	switch t.kind {
	case String, Key, Number:
		return fmt.Sprintf("{Kind: %q, Value: %q}", t.kind, t.value)
	case Boolean:
		return fmt.Sprintf("{Kind: %q, ValueBoolean: %t}", t.kind, t.valueBool)
	case Null:
		return fmt.Sprintf("{Kind: %q}", t.kind)
	case Delimiter:
		return fmt.Sprintf("{Kind: %q, KindDelimiter: %q}", t.kind, t.valueDelimiter)
	case EOF, Unknown:
		return fmt.Sprintf("{Kind: %q}", t.kind)
	default:
		panic(fmt.Sprintf("invalid token kind: %d", t.kind))
	}
}
