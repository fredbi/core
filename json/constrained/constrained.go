package constrained

import (
	"fmt"
	"io"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/lexers"
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/nodes"
	codes "github.com/fredbi/core/json/nodes/error-codes"
	"github.com/fredbi/core/json/nodes/light"
)

// predefined constrained documents (e.g. used by jsonschema)
// using hooks to customize light.Node.

// MakeObject build an [Object].
func MakeObject(opts ...json.Option) Object {
	return Object{
		Document: json.Make(opts...),
	}
}

// Object is a [json.Document] constrained to be a JSON object.
type Object struct {
	json.Document
	isObject bool
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

func (d *Object) Hooks() light.DecodeOptions {
	return d.hooks()
}

func (d *Object) hooks() light.DecodeOptions {
	decodeOptions := d.DecodeOptions
	decodeOptions.NodeHook = d.mustBeObject

	return decodeOptions
}

func (d *Object) mustBeObject(l lexers.Lexer, tok token.T) (skip bool, err error) {
	if d.isObject {
		return false, nil
	}

	if l.IndentLevel() == 1 && tok.IsStartObject() {
		d.isObject = true

		return false, nil
	}

	return false, fmt.Errorf("a JSON object is expected. Got: %v: %w", tok, codes.ErrNode)
}

func (d *Object) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = d.Store()
	context.DO = d.hooks()
	d.isObject = false

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

// Array is a [json.Document] constrained to be a JSON array.
type Array struct {
	json.Document
	isArray bool
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

func (d *Array) Hooks() light.DecodeOptions {
	return d.hooks()
}

func (d *Array) hooks() light.DecodeOptions {
	decodeOptions := d.DecodeOptions
	decodeOptions.NodeHook = d.mustBeArray

	return decodeOptions
}

func (d *Array) mustBeArray(l lexers.Lexer, tok token.T) (skip bool, err error) {
	if d.isArray {
		return false, nil
	}

	if l.IndentLevel() == 1 && tok.IsStartArray() {
		d.isArray = true

		return false, nil
	}

	return false, fmt.Errorf("a JSON array is expected. Got: %v: %w", tok, codes.ErrNode)
}

func (d *Array) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = d.Store()
	context.DO = d.hooks()
	d.isArray = false

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

// StringOrArrayOfStrings is a [json.Document] constrained to be either a string or an array of strings.
type StringOrArrayOfStrings struct {
	json.Document
	isStringOrArrayOfStrings bool
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

func (d *StringOrArrayOfStrings) Hooks() light.DecodeOptions {
	return d.hooks()
}

func (d *StringOrArrayOfStrings) hooks() light.DecodeOptions {
	decodeOptions := d.DecodeOptions
	decodeOptions.NodeHook = d.mustBeStringOrArrayOfStrings

	return decodeOptions
}

func (d *StringOrArrayOfStrings) mustBeStringOrArrayOfStrings(
	l lexers.Lexer,
	tok token.T,
) (skip bool, err error) {
	if d.isStringOrArrayOfStrings {
		return false, nil
	}

	level := l.IndentLevel()
	switch {
	case level == 0 && tok.Kind() == token.String:
		fallthrough
	case level == 0 && tok.IsEndArray():
		d.isStringOrArrayOfStrings = true

		return false, nil
	case level == 1 && tok.IsStartArray():
		fallthrough
	case level == 1 && (tok.Kind() == token.String || tok.IsComma()):

		return false, nil
	default:
		return false, fmt.Errorf(
			"a string or an array of strings is expected. Got: %v: %w",
			tok,
			codes.ErrNode,
		)
	}
}

func (d *StringOrArrayOfStrings) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = d.Store()
	context.DO = d.hooks()
	d.isStringOrArrayOfStrings = false

	n := d.Node()
	n.Decode(context)
	light.RedeemParentContext(context)

	return lex.Err()
}

func MakeBoolOrObject(opts ...json.Option) BoolOrObject {
	return BoolOrObject{
		Document: json.Make(opts...),
	}
}

// BoolOrObject is a [json.Document] constrained to be either a boolean or an object.
type BoolOrObject struct {
	json.Document
	isBoolOrObject bool
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

func (d *BoolOrObject) Hooks() light.DecodeOptions {
	return d.hooks()
}

func (d *BoolOrObject) hooks() light.DecodeOptions {
	decodeOptions := d.DecodeOptions
	decodeOptions.NodeHook = d.mustBeBoolOrObject

	return decodeOptions
}

func (d *BoolOrObject) mustBeBoolOrObject(l lexers.Lexer, tok token.T) (skip bool, err error) {
	if d.isBoolOrObject {
		return false, nil
	}

	switch {
	case l.IndentLevel() == 0 && tok.Kind() == token.Boolean:
		fallthrough
	case l.IndentLevel() == 1 && tok.IsStartObject():
		d.isBoolOrObject = true

		return false, nil
	default:
		return false, fmt.Errorf(
			"a boolean or an object is expected. Got: %v: %w",
			tok,
			codes.ErrNode,
		)
	}
}

func (d *BoolOrObject) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = d.Store()
	context.DO = d.hooks()
	d.isBoolOrObject = false

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

// ObjectOrArray is a [json.Document] constrained to be either an object or an array of objects.
type ObjectOrArrayOfObjects struct {
	json.Document
	isObject bool
	isArray  bool
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

func (d *ObjectOrArrayOfObjects) Hooks() light.DecodeOptions {
	return d.hooks()
}

func (d *ObjectOrArrayOfObjects) hooks() light.DecodeOptions {
	decodeOptions := d.DecodeOptions
	decodeOptions.NodeHook = d.mustBeObjectOrArrayOfObjects
	decodeOptions.AfterElem = d.elementMustBeObject

	return decodeOptions
}

func (d *ObjectOrArrayOfObjects) mustBeObjectOrArrayOfObjects(
	l lexers.Lexer,
	tok token.T,
) (skip bool, err error) {
	if d.isObject || d.isArray {
		return false, nil
	}

	switch {
	case l.IndentLevel() == 0 && tok.IsEndArray():
		fallthrough
	case l.IndentLevel() == 1 && tok.IsStartArray():
		d.isArray = true
	case l.IndentLevel() == 0 && tok.IsEndObject():
		fallthrough
	case l.IndentLevel() == 1 && tok.IsStartObject():
		d.isObject = true
	case l.IndentLevel() == 0:
		return false, fmt.Errorf(
			"an object or an array of objects is expected. Got: %v: %w",
			tok,
			codes.ErrNode,
		)
	default:
	}

	return false, nil
}

func (d *ObjectOrArrayOfObjects) elementMustBeObject(
	l lexers.Lexer,
	elem light.Node,
) (skip bool, err error) {
	if d.isObject || l.IndentLevel() > 2 {
		return false, nil
	}

	if d.isArray && elem.Kind() == nodes.KindObject {
		return false, nil
	}

	return false, fmt.Errorf(
		"an object or an array of objects is expected. Got array element: %v: %w",
		elem.Kind(),
		codes.ErrNode,
	)
}

func (d *ObjectOrArrayOfObjects) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = d.Store()
	context.DO = d.hooks()
	d.isObject = false
	d.isArray = false

	n := d.Node()
	n.Decode(context)
	light.RedeemParentContext(context)

	return lex.Err()
}
