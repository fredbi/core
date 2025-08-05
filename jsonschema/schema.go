package jsonschema

import (
	"fmt"
	"io"
	"iter"
	"strings"

	"github.com/fredbi/core/json"
	"github.com/fredbi/core/json/lexers"
	"github.com/fredbi/core/json/lexers/token"
	codes "github.com/fredbi/core/json/nodes/error-codes"
	"github.com/fredbi/core/json/nodes/light"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/jsonschema/analyzers"
	"github.com/fredbi/core/jsonschema/meta"
)

// SchemaType is one of the seven JSON schema simple types,
// which may be referenced in a "type" validation.
type SchemaType uint8

const (
	SchemaTypeNull SchemaType = iota
	SchemaTypeObject
	SchemaTypeArray
	SchemaTypeString
	SchemaTypeNumber
	SchemaTypeInteger // >= draft4: a number with a 0 fractional part is now considered an integer
	SchemaTypeBool
)

var EmptySchema = Schema{}

// Schema represents a valid JSON schema specification.
//
// [Schema] supports several JSON schema dialects (or versions).
//
// A [Schema] extends a [json.Document], so everything you can do on a [json.Document] works on a [Schema] as well.
type Schema struct {
	json.Document
	*options

	defined    bool
	core       Core
	applicator Applicator
	validation Validation
	metadata   Metadata
	extensions analyzers.Extensions
	extras     []*light.Node
}

// Make builds an empty JSON schema.
func Make(opts ...Option) Schema {
	o := optionsWithDefaults(opts)
	return Schema{
		Document: json.Make(o.documentOptions...),
		options:  o,
	}
}

// New builds a pointer to an empty JSON schema.
func New(opts ...Option) *Schema {
	s := Make(opts...)

	return &s
}

// IsDefined indicates if the [Schema] is defined.
func (s Schema) IsDefined() bool {
	return s.defined
}

// Core definitions for this [Schema].
//
// See https://json-schema.org/draft/2020-12/meta/core
// and
// https://json-schema.org/draft/2020-12/meta/content
func (s Schema) Core() Core {
	return Core{}
}

// HasApplicator tells if there is a non-empty [Applicator] for this [Schema].
func (s Schema) HasApplicator() bool {
	return s.applicator.IsDefined()
}

// Applicator definitions for this [Schema].
//
// See https://json-schema.org/draft/2020-12/meta/applicator
// and
// https://json-schema.org/draft/2020-12/meta/unevaluated
func (s Schema) Applicator() Applicator {
	return Applicator{}
}

// HasMetadata tells if there is a non-empty [meta.Data] for this [Schema].
func (s Schema) HasMetadata() bool {
	return false
}

// Metadata definitions for this [Schema].
//
// See https://json-schema.org/draft/2020-12/meta/meta-data
func (s Schema) Metadata() meta.Data {
	// title, description, examples, $deprecated, $id, readOnly, writeOnly, $comment...
	return meta.Data{}
}

// HasValidation tells if there is a non-empty [Validation] for this [Schema].
func (s Schema) HasValidation() bool {
	return s.validation.IsDefined()
}

// Validation definitions for this [Schema].
//
// See https://json-schema.org/draft/2020-12/meta/validation
// and
// https://json-schema.org/draft/2020-12/meta/format-assertion
func (s Schema) Validation() Validation {
	return s.validation
}

// HasExtensions tells if there are some "x-*" extension keys for this [Schema].
func (s Schema) HasExtensions() bool {
	return len(s.extensions) > 0
}

// Extensions returns "x-*" extensions as a map of [json.Document] s.
func (s Schema) Extensions() analyzers.Extensions {
	return s.extensions
}

// HasExtraKeys tells if there are some extra non-recognized keys for this [Schema].
func (s Schema) HasExtraKeys() bool {
	return len(s.extras) > 0
}

// ExtraKeys returns non-recognized keys.
func (s Schema) ExtraKeys() iter.Seq2[string, json.Document] {
	// other extra keys not recognized as extensions
	return nil
}

func (s *Schema) Decode(r io.Reader) error {
	lex, redeem := s.LexerFromReaderFactory()(r)
	defer redeem()

	return s.decode(lex)
}

func (s *Schema) UnmarshalJSON(data []byte) error {
	lex, redeem := s.LexerFactory()(data)
	defer redeem()

	return s.decode(lex)
}

func (s *Schema) decode(lex lexers.Lexer) error {
	context := light.BorrowParentContext()
	context.L = lex
	context.S = s.Store()
	context.DO = s.hooks()
	octx := poolOfOverlayContexts.Borrow()
	octx.initialLevel = lex.IndentLevel()
	context.X = octx

	n := s.Node()
	n.Decode(context)
	light.RedeemParentContext(context)
	poolOfOverlayContexts.Redeem(octx)

	return lex.Err()
}

func (s *Schema) hooks() light.DecodeOptions {
	decodeOptions := s.DecodeOptions
	decodeOptions.NodeHook = s.mustBeBoolOrObject
	decodeOptions.BeforeKey = s.beforeKey

	return decodeOptions
}

type schemaContext struct {
	initialLevel   int
	isBoolOrObject bool

	anchor stores.Handle
}

func (s *Schema) mustBeBoolOrObject(
	ctx *light.ParentContext,
	l lexers.Lexer,
	tok token.T,
) (skip bool, err error) {
	octx, ok := ctx.X.(*schemaContext)
	if !ok {
		return false, nil
	}

	if octx.isBoolOrObject {
		return false, nil
	}
	level := l.IndentLevel() - octx.initialLevel

	switch {
	case level == 0 && tok.Kind() == token.Boolean:
		fallthrough
	case level == 1 && tok.IsStartObject():
		octx.isBoolOrObject = true

		return false, nil
	default:
		return false, fmt.Errorf(
			"a boolean or an object is expected. Got: %v: %w",
			tok,
			codes.ErrNode,
		)
	}
}

func (s *Schema) beforeKey(ctx *light.ParentContext, l lexers.Lexer, key values.InternedKey) (skip bool, err error) {
	// TODO
	if _, isCore := coreKeys[key]; isCore {
		err := s.core.decode(ctx, key)
		return false, err
	}

	if _, isApplicator := applicatorKeys[key]; isApplicator {
		err := s.applicator.decode(ctx, key, &s.core.version)
		return false, err
	}

	if _, isValidation := validationKeys[key]; isValidation {
		err := s.validation.decode(ctx, key, &s.core.version)
		return false, err
	}

	if _, isMetadata := metadataKeys[key]; isMetadata {
		err := s.metadata.decode(ctx, key, &s.core.version)
		return false, err
	}

	// extensions
	if ext := key.String(); strings.HasPrefix(ext, "x-") {
		if s.extensions == nil {
			s.extensions = make(analyzers.Extensions)
		}
		// consume node
		// switch context
		var n light.Node
		n.Decode(ctx)
		// switch back context
		doc := json.NewBuilder(s.Store()).WithRoot(n).Document() // TODO: pool
		s.extensions.Add(ext, doc)

		return
	}

	// extra key
	// TODO

	return false, nil
}
