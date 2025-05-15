package constrained

import (
	"fmt"
	"io"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/lexers"
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/nodes/light"
)

// predefined constrained documents (e.g. used by jsonschema)
// using hooks to customize light.Node.

func MakeObject(opts ...json.Option) Object {
	return Object{
		Document: json.Make(opts...),
	}
}

// Object is a [Document] constrained to be a JSON object.
type Object struct {
	json.Document
}

func (d *Object) Decode(r io.Reader) error {
	lex, redeem := d.LexerFromReaderFactory()(r)
	defer redeem()

	return d.decode(lex)
}
func (d *Object) UnmarshalJSON(data []byte) error {
	lex, redeem := d.LexerFactory()(data)
	defer redeem()

	return d.decode(lex)
}

func (d Object) hooks() light.DecodeOptions {
	decodeOptions := d.DecodeOptions
	decodeOptions.NodeHook = mustBeObject

	return decodeOptions
}

func (d *Object) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = d.Store()
	context.DO = d.hooks()

	n := d.Node()
	n.Decode(context)
	light.RedeemParentContext(context)

	return lex.Err()
}

func MakeArray(opts ...json.Option) Array {
	return Array{
		Document: json.Make(opts...),
	}
}

// Array is a [Document] constrained to be a JSON array.
type Array struct {
	json.Document
}

func (d *Array) Decode(r io.Reader) error {
	lex, redeem := d.LexerFromReaderFactory()(r)
	defer redeem()

	return d.decode(lex)
}

func (d *Array) UnmarshalJSON(data []byte) error {
	lex, redeem := d.LexerFactory()(data)
	defer redeem()

	return d.decode(lex)
}

func (d Array) hooks() light.DecodeOptions {
	decodeOptions := d.DecodeOptions
	decodeOptions.NodeHook = mustBeArray

	return decodeOptions
}

func (d *Array) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = d.Store()
	context.DO = d.hooks()
	n := d.Node()
	n.Decode(context)
	light.RedeemParentContext(context)

	return lex.Err()
}

func MakeStringOrArrayOfStrings(opts ...json.Option) StringOrArrayOfStrings {
	return StringOrArrayOfStrings{
		Document: json.Make(opts...),
	}
}

// StringOrArrayOfStrings is a [Document] constrained to be either a string or an array of strings.
type StringOrArrayOfStrings struct {
	json.Document
}

func (d *StringOrArrayOfStrings) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = d.Store()
	context.DO = d.hooks()
	n := d.Node()
	n.Decode(context)
	light.RedeemParentContext(context)

	return lex.Err()
}

func (d *StringOrArrayOfStrings) Decode(r io.Reader) error {
	lex, redeem := d.LexerFromReaderFactory()(r)
	defer redeem()

	return d.decode(lex)
}
func (d *StringOrArrayOfStrings) UnmarshalJSON(data []byte) error {
	lex, redeem := d.LexerFactory()(data)
	defer redeem()

	return d.decode(lex)
}
func (d StringOrArrayOfStrings) hooks() light.DecodeOptions {
	decodeOptions := d.DecodeOptions
	decodeOptions.NodeHook = mustBeStringOrArrayOfStrings

	return decodeOptions
}

func MakeBoolOrObject(opts ...json.Option) BoolOrObject {
	return BoolOrObject{
		Document: json.Make(opts...),
	}
}

// BoolOrObject is a [Document] constrained to be either a boolean or an object.
type BoolOrObject struct {
	json.Document
}

func (d *BoolOrObject) Decode(r io.Reader) error {
	lex, redeem := d.LexerFromReaderFactory()(r)
	defer redeem()

	return d.decode(lex)
}
func (d *BoolOrObject) UnmarshalJSON(data []byte) error {
	lex, redeem := d.LexerFactory()(data)
	defer redeem()

	return d.decode(lex)
}

func (d BoolOrObject) hooks() light.DecodeOptions {
	decodeOptions := d.DecodeOptions
	decodeOptions.NodeHook = mustBeBoolOrObject

	return decodeOptions
}

func (d *BoolOrObject) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = d.Store()
	context.DO = d.hooks()
	n := d.Node()
	n.Decode(context)
	light.RedeemParentContext(context)

	return lex.Err()
}

func MakeObjectOrArrayOfObjects(opts ...json.Option) ObjectOrArrayOfObjects {
	return ObjectOrArrayOfObjects{
		Document: json.Make(opts...),
	}
}

// ObjectOrArray is a [Document] constrained to be either an object or an array of objects.
// TODO
type ObjectOrArrayOfObjects struct {
	json.Document
}

func (d *ObjectOrArrayOfObjects) Decode(r io.Reader) error {
	lex, redeem := d.LexerFromReaderFactory()(r)
	defer redeem()

	return d.decode(lex)
}
func (d *ObjectOrArrayOfObjects) UnmarshalJSON(data []byte) error {
	lex, redeem := d.LexerFactory()(data)
	defer redeem()

	return d.decode(lex)
}

func (d ObjectOrArrayOfObjects) hooks() light.DecodeOptions {
	decodeOptions := d.DecodeOptions
	decodeOptions.NodeHook = mustBeObjectOrArrayOfObjects

	return decodeOptions
}

func (d *ObjectOrArrayOfObjects) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = d.Store()
	context.DO = d.hooks()
	n := d.Node()
	n.Decode(context)
	light.RedeemParentContext(context)

	return lex.Err()
}

func mustBeObject(l lexers.Lexer, tok token.T) (skip bool, err error) {
	if l.IndentLevel() > 0 {
		return false, nil
	}

	if tok.IsStartObject() {
		return false, nil
	}

	var delim string
	if tok.IsDelimiter() {
		delim = ": " + tok.Delimiter().String()
	}
	return false, fmt.Errorf("a JSON object is expected. Got: %v%s", tok, delim)
}

func mustBeArray(l lexers.Lexer, tok token.T) (skip bool, err error) {
	if l.IndentLevel() > 0 {
		return false, nil
	}

	if tok.IsStartArray() {
		return false, nil
	}

	var delim string
	if tok.IsDelimiter() {
		delim = ": " + tok.Delimiter().String()
	}
	return false, fmt.Errorf("a JSON array is expected. Got: %v%s", tok, delim)
}

func mustBeStringOrArrayOfStrings(l lexers.Lexer, tok token.T) (skip bool, err error) {
	switch {
	case l.IndentLevel() == 0 && tok.Kind() == token.String:
		return false, nil
	case l.IndentLevel() == 0 && tok.IsStartArray():
		return false, nil
	case l.IndentLevel() == 1 && (tok.IsEndArray() || tok.Kind() == token.String):
		return false, nil
	default:
		var delim string
		if tok.IsDelimiter() {
			delim = ": " + tok.Delimiter().String()
		}
		return false, fmt.Errorf("a string or an array of strings is expected. Got: %v%s", tok, delim)
	}
}

func mustBeBoolOrObject(l lexers.Lexer, tok token.T) (skip bool, err error) {
	switch {
	case l.IndentLevel() == 0 && tok.Kind() == token.Boolean:
		return false, nil
	case l.IndentLevel() == 0 && tok.IsStartObject():
		return false, nil
	default:
		var delim string
		if tok.IsDelimiter() {
			delim = ": " + tok.Delimiter().String()
		}
		return false, fmt.Errorf("a boolean or an object is expected. Got: %v%s", tok, delim)
	}
}

func mustBeObjectOrArrayOfObjects(l lexers.Lexer, tok token.T) (skip bool, err error) {
	switch {
	case l.IndentLevel() == 0 && tok.Clone().IsStartArray():
		return false, nil
	case l.IndentLevel() == 0 && tok.IsStartObject():
		return false, nil
	case l.IndentLevel() == 1 && (tok.IsEndArray() || tok.IsStartObject()):
		return false, nil
	default:
		var delim string
		if tok.IsDelimiter() {
			delim = ": " + tok.Delimiter().String()
		}
		return false, fmt.Errorf("an object or an array of objects is expected. Got: %v%s", tok, delim)
	}
}
