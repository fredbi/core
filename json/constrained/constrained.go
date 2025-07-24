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

type objectContext struct {
	initialLevel int
	isObject     bool
}

func (c *objectContext) Reset() {
	c.initialLevel = 0
	c.isObject = false
}

// Object is a [json.Document] constrained to be a JSON object.
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

func (d *Object) Hooks() light.DecodeOptions {
	return d.hooks()
}

func (d *Object) hooks() light.DecodeOptions {
	decodeOptions := d.DecodeOptions
	decodeOptions.NodeHook = d.mustBeObject

	return decodeOptions
}

func (d *Object) mustBeObject(
	ctx *light.ParentContext,
	l lexers.Lexer,
	tok token.T,
) (skip bool, err error) {
	octx, ok := ctx.X.(*objectContext)
	if !ok {
		return false, nil
	}

	if octx.isObject {
		return false, nil
	}

	level := l.IndentLevel() - octx.initialLevel

	if level == 1 && tok.IsStartObject() {
		octx.isObject = true

		return false, nil
	}

	return false, fmt.Errorf("a JSON object is expected. Got: %v: %w", tok, codes.ErrNode)
}

func (d *Object) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = d.Store()
	context.DO = d.hooks()
	octx := poolOfObjectContexts.Borrow()
	octx.initialLevel = lex.IndentLevel()
	context.X = octx

	n := d.Node()
	n.Decode(context)
	light.RedeemParentContext(context)
	poolOfObjectContexts.Redeem(octx)

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
}

type arrayContext struct {
	initialLevel int
	isArray      bool
}

func (c *arrayContext) Reset() {
	c.initialLevel = 0
	c.isArray = false
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

func (d *Array) mustBeArray(
	ctx *light.ParentContext,
	l lexers.Lexer,
	tok token.T,
) (skip bool, err error) {
	octx, ok := ctx.X.(*arrayContext)
	if !ok {
		return false, nil
	}

	if octx.isArray {
		return false, nil
	}

	level := l.IndentLevel() - octx.initialLevel
	if level == 1 && tok.IsStartArray() {
		octx.isArray = true

		return false, nil
	}

	return false, fmt.Errorf("a JSON array is expected. Got: %v: %w", tok, codes.ErrNode)
}

func (d *Array) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = d.Store()
	context.DO = d.hooks()
	octx := poolOfArrayContexts.Borrow()
	octx.initialLevel = lex.IndentLevel()
	context.X = octx

	n := d.Node()
	n.Decode(context)
	light.RedeemParentContext(context)
	poolOfArrayContexts.Redeem(octx)

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
}

type stringOfArrayContext struct {
	initialLevel             int
	isStringOrArrayOfStrings bool
}

func (c *stringOfArrayContext) Reset() {
	c.initialLevel = 0
	c.isStringOrArrayOfStrings = false
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
	ctx *light.ParentContext,
	l lexers.Lexer,
	tok token.T,
) (skip bool, err error) {
	octx, ok := ctx.X.(*stringOfArrayContext)
	if !ok {
		return false, nil
	}
	if octx.isStringOrArrayOfStrings {
		return false, nil
	}

	level := l.IndentLevel() - octx.initialLevel
	switch {
	case level == 0 && tok.Kind() == token.String:
		fallthrough
	case level == 0 && tok.IsEndArray():
		octx.isStringOrArrayOfStrings = true

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
	octx := poolOfStringOrArrayContexts.Borrow()
	octx.initialLevel = lex.IndentLevel()
	context.X = octx

	n := d.Node()
	n.Decode(context)
	light.RedeemParentContext(context)
	poolOfStringOrArrayContexts.Redeem(octx)

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
}

type boolOrObjectContext struct {
	initialLevel   int
	isBoolOrObject bool
}

func (c *boolOrObjectContext) Reset() {
	c.initialLevel = 0
	c.isBoolOrObject = false
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

func (d *BoolOrObject) mustBeBoolOrObject(
	ctx *light.ParentContext,
	l lexers.Lexer,
	tok token.T,
) (skip bool, err error) {
	boolOrObjectContext, ok := ctx.X.(*boolOrObjectContext)
	if !ok {
		return false, nil
	}

	if boolOrObjectContext.isBoolOrObject {
		return false, nil
	}
	level := l.IndentLevel() - boolOrObjectContext.initialLevel

	switch {
	case level == 0 && tok.Kind() == token.Boolean:
		fallthrough
	case level == 1 && tok.IsStartObject():
		boolOrObjectContext.isBoolOrObject = true

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
	octx := poolOfBoolOrObjectContexts.Borrow()
	octx.initialLevel = lex.IndentLevel()
	context.X = octx

	n := d.Node()
	n.Decode(context)
	light.RedeemParentContext(context)
	poolOfBoolOrObjectContexts.Redeem(octx)

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
}

type objectOrArrayOfObjectsContext struct {
	initialLevel int
	isObject     bool
	isArray      bool
}

func (c *objectOrArrayOfObjectsContext) Reset() {
	c.initialLevel = 0
	c.isObject = false
	c.isArray = false
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
	ctx *light.ParentContext,
	l lexers.Lexer,
	tok token.T,
) (skip bool, err error) {
	octx, ok := ctx.X.(*objectOrArrayOfObjectsContext)
	if !ok {
		return false, nil
	}
	if octx.isObject || octx.isArray {
		return false, nil
	}

	level := l.IndentLevel() - octx.initialLevel
	switch {
	case level == 0 && tok.IsEndArray():
		fallthrough
	case level == 1 && tok.IsStartArray():
		octx.isArray = true
	case level == 0 && tok.IsEndObject():
		fallthrough
	case level == 1 && tok.IsStartObject():
		octx.isObject = true
	case level == 0:
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
	ctx *light.ParentContext,
	l lexers.Lexer,
	elem light.Node,
) (skip bool, err error) {
	octx, ok := ctx.X.(*objectOrArrayOfObjectsContext)
	if !ok {
		return false, nil
	}
	if octx.isObject || l.IndentLevel()-octx.initialLevel > 2 {
		return false, nil
	}

	if octx.isArray && elem.Kind() == nodes.KindObject {
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
	octx := poolOfObjectOrArrayOfObjectsContexts.Borrow()
	octx.initialLevel = lex.IndentLevel()
	context.X = octx

	n := d.Node()
	n.Decode(context)
	light.RedeemParentContext(context)
	poolOfObjectOrArrayOfObjectsContexts.Redeem(octx)

	return lex.Err()
}
