package jsonschema

import (
	"fmt"
	"io"
	"iter"
	"slices"
	"strings"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/jsonpath"
	"github.com/fredbi/core/json/lexers"
	"github.com/fredbi/core/json/lexers/token"
	codes "github.com/fredbi/core/json/nodes/error-codes"
	"github.com/fredbi/core/json/nodes/light"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/jsonschema/analyzers"
	"github.com/fredbi/core/jsonschema/overlay"
)

// Overlay is an overlay specification for a JSON schema.
//
// It mostly clones the overlay specification published by OpenAPI
// (https://spec.openapis.org/overlay/v1.0.0.html),
// but does not require title, version or any metadata.
//
// Unlike the OpenAPI overlay, a schema [Overlay] may be empty or contain an empty list of actions.
//
// Actions must specify a valid [jsonpath.Expression].
type Overlay struct {
	*overlayOptions
	json.Document

	finder         *jsonpath.PathFinder
	overlayVersion overlay.Version
	info           overlay.Info
	extends        stores.Handle
	actions        []overlay.Action
	extensions     analyzers.Extensions
}

// MakeOverlay builds a JSON schema [Overlay].
func MakeOverlay(opts ...OverlayOption) Overlay {
	o := overlayOptionsWithDefaults(opts) // TODO: borrow from pool + finalizer
	return Overlay{
		overlayOptions: o,
		finder:         jsonpath.New(), // TODO: use pool
		Document:       json.Make(o.documentOptions...),
	}
}

func (o *Overlay) Reset() {
	// TODO: reset options and finder
	o.overlayVersion = overlay.VersionUndefined
	o.info.Reset()
	o.extends = stores.HandleZero
	o.actions = o.actions[:0]
}

// Version of the Overlay schema.
//
// Ex: 1.0.0
func (o Overlay) Version() overlay.Version {
	return o.overlayVersion
}

// Info provides metadata about the [Overlay].
func (o Overlay) Info() overlay.Info {
	return o.info
}

// Extends is a URI reference that identifies the target document this overlay applies to.
func (o Overlay) Extends() string {
	return o.Store().Get(o.extends).String()
}

// Action returns an ordered list of actions to be applied to the target document.
//
// Unlike openapi Overlays, the list may be empty.
func (o Overlay) Actions() iter.Seq[overlay.Action] {
	return slices.Values(o.actions)
}

func (o Overlay) Extensions() analyzers.Extensions {
	return o.extensions
}

// ApplyTo applies the set of actions defined by the [Overlay] to a [Schema].
//
// If no change applies to the input schema, the input is returned unaltered.
//
// If the target expression resolves as an object and the update specification is not an object,
// the action rule is ignored and the input is returned unaltered (TODO: this form of error handling is questionable).
func (o Overlay) ApplyTo(sch Schema) Schema {
	b := NewBuilder().From(sch) // TODO: pool
	lastCorrect := sch

	for _, action := range o.actions {
		for pointer := range o.finder.Pointers(sch.Document, action.Target()) {
			b = b.AtPointerMerge(pointer, action.Update()) // TODO: implement this
			if !b.Ok() {
				return lastCorrect
			}

			lastCorrect = b.Schema()
		}
	}

	return lastCorrect
}

func (o *Overlay) Decode(r io.Reader) error {
	lex, redeem := o.LexerFromReaderFactory()(r)
	defer redeem()

	return o.decode(lex)
}

func (o *Overlay) UnmarshalJSON(data []byte) error {
	lex, redeem := o.LexerFactory()(data)
	defer redeem()

	return o.decode(lex)
}

func (o *Overlay) hooks() light.DecodeOptions {
	decodeOptions := o.DecodeOptions
	decodeOptions.NodeHook = o.mustBeObject
	decodeOptions.AfterKey = o.afterKey // TODO: we lose the error path, perhaps we should to beforeKey

	return decodeOptions
}

var (
	overlayKey = values.MakeInternedKey("overlay")
	infoKey    = values.MakeInternedKey("info")
	extendsKey = values.MakeInternedKey("extends")
	actionsKey = values.MakeInternedKey("actions")
)

func (o *Overlay) decodeActionsArray(n light.Node) error {
	if !n.IsArray() {
		return fmt.Errorf("actions should be an array: %w", overlay.ErrOverlay)
	}

	o.actions = make([]overlay.Action, 0, n.Len())
	s := o.Store()

	for e := range n.Elems() {
		if !e.IsObject() {
			return fmt.Errorf("action element should be an object: %w", overlay.ErrOverlay)
		}

		var action overlay.Action
		if err := action.Decode(s, e); err != nil {
			return err
		}

		o.actions = append(o.actions, action)
	}

	return nil
}

func (o *Overlay) afterKey(
	ctx *light.ParentContext,
	l lexers.Lexer,
	key values.InternedKey,
	n light.Node,
) (skip bool, err error) {
	s := o.Store()

	switch key {
	case overlayKey:
		var version overlay.Version
		if err := version.Decode(s, n); err != nil {
			return false, err
		}
		o.overlayVersion = version

	case infoKey:
		var info overlay.Info
		if err := info.Decode(s, n); err != nil {
			return false, err
		}
		o.info = info
	case extendsKey:
		if !n.IsString(s) {
			return false, fmt.Errorf("extends should be a string:%w", overlay.ErrOverlay)
		}
		// TODO: validate URI? does not seem to be a strict requirement
		o.extends, _ = n.Handle()
	case actionsKey:
		if err := o.decodeActionsArray(n); err != nil {
			return false, err
		}
	default:
		// x-* extensions
		if ext := key.String(); strings.HasPrefix(ext, "x-") {
			if o.extensions == nil {
				o.extensions = make(analyzers.Extensions)
			}
			doc := json.NewBuilder(s).WithRoot(n).Document() // TODO: pool
			o.extensions.Add(ext, doc)
		}
	}

	// other keys remain part of the document, but uninterpreted
	return false, nil
}

func (o *Overlay) mustBeObject(
	// TODO: generic constrained.MustBeObjectHook[overlayContext]???
	ctx *light.ParentContext,
	l lexers.Lexer,
	tok token.T,
) (skip bool, err error) {
	octx, ok := ctx.X.(*overlayContext)
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

type overlayContext struct {
	// TODO: factorize constrained.ObjectContext
	initialLevel int
	isObject     bool
}

func (o *Overlay) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = o.Store()
	context.DO = o.hooks()
	octx := poolOfOverlayContexts.Borrow()
	octx.initialLevel = lex.IndentLevel()
	context.X = octx

	n := o.Node()
	n.Decode(context)
	light.RedeemParentContext(context)
	poolOfOverlayContexts.Redeem(octx)

	return lex.Err()
}
