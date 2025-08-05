package jsonschema

import (
	"iter"

	"github.com/fredbi/core/json/nodes/light"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/stores/values"
)

func schemaFromNode(s stores.Store, n *light.Node) Schema {
	_ = s                      // TODO
	b := NewBuilder( /* s */ ) // TODO: borrow

	return b.WithRoot(*n).Schema()
}

// Core jsonschema definitions.
//
// This includes the following JSON schema keywords:
//
//   - $anchor (>= draft 2019)
//   - $defs (>= draft 2019 ; for pre-draft 2019: definitions)
//   - $dynamicAnchor: draft 2020
//   - $dynamicRef: draft 2020
//   - $id (for pre-draft 6: id ; >= draft 2019: no fragment allowed)
//   - $ref
//   - $schema
//   - $recursiveAnchor, $recursiveRef (= draft 2019)
//
// Content vocabulary:
//
//   - contentEncoding (>= draft 7)
//   - contentMediaType (>= draft 7)
type Core struct {
	s             stores.Store
	anchor        stores.Handle
	isBool        bool // >= draft 6
	boolValue     bool
	defs          []*light.Node
	defsIndex     map[values.InternedKey]int
	dynamicAnchor stores.Handle
	hasRef        bool
	defined       bool
	id            stores.Handle
	ref           Ref
	schemaID      stores.Handle

	version VersionRequirements
}

var (
	anchorKey           = values.MakeInternedKey("$anchor")
	contentEncodingKey  = values.MakeInternedKey("contentEncoding")
	contentMediaTypeKey = values.MakeInternedKey("contentMediaType")
	definitionsKey      = values.MakeInternedKey("definitions")
	defsKey             = values.MakeInternedKey("$defs")
	draft4IDKey         = values.MakeInternedKey("id")
	dynamicAnchorKey    = values.MakeInternedKey("$dynamicAnchor")
	dynamicRefKey       = values.MakeInternedKey("$dynamicRef")
	idKey               = values.MakeInternedKey("$id")
	recursiveAnchorKey  = values.MakeInternedKey("recursiveAnchor")
	recursiveRefKey     = values.MakeInternedKey("recursiveRef")
	refKey              = values.MakeInternedKey("$ref")
	schemaKey           = values.MakeInternedKey("$schema")

	coreKeys = map[values.InternedKey]struct{}{
		anchorKey:           {},
		contentEncodingKey:  {},
		contentMediaTypeKey: {},
		definitionsKey:      {},
		defsKey:             {},
		draft4IDKey:         {},
		dynamicAnchorKey:    {},
		dynamicRefKey:       {},
		idKey:               {},
		recursiveAnchorKey:  {},
		recursiveRefKey:     {},
		refKey:              {},
		schemaKey:           {},
	}

	coreConstraints = map[values.InternedKey]VersionRequirements{
		draft4IDKey:         {MaxVersion: VersionDraft5, StrictMaxVersion: VersionDraft5},
		idKey:               {MinVersion: VersionDraft6},
		definitionsKey:      {MaxVersion: VersionDraft7},
		contentEncodingKey:  {MinVersion: VersionDraft7},
		contentMediaTypeKey: {MinVersion: VersionDraft7},
		anchorKey:           {MinVersion: VersionDraft2019},
		defsKey:             {MinVersion: VersionDraft2019},
		recursiveAnchorKey:  {MinVersion: VersionDraft2019, MaxVersion: VersionDraft2019, StrictMaxVersion: VersionDraft2019},
		recursiveRefKey:     {MinVersion: VersionDraft2019, MaxVersion: VersionDraft2019, StrictMaxVersion: VersionDraft2019},
		dynamicAnchorKey:    {MinVersion: VersionDraft2020},
		dynamicRefKey:       {MinVersion: VersionDraft2020},
	}
)

// IsTrue indicates if the schema is specified as a boolean true value.
func (c Core) IsTrue() bool {
	return c.isBool && c.boolValue
}

// IsFalse indicates if the schema is specified as a boolean false value.
func (c Core) IsFalse() bool {
	return c.isBool && !c.boolValue
}

func (c Core) ID() string {
	if c.id == stores.HandleZero {
		return ""
	}

	return c.s.Get(c.id).String()
}

func (c Core) SchemaID() string { // TODO: URI + version
	if c.schemaID == stores.HandleZero {
		return ""
	}

	return c.s.Get(c.schemaID).String()
}

func (c Core) Anchor() string {
	if c.anchor == stores.HandleZero {
		return ""
	}

	return c.s.Get(c.anchor).String()
}

func (c Core) DynamicAnchor() string {
	if c.dynamicAnchor == stores.HandleZero {
		return ""
	}

	return c.s.Get(c.dynamicAnchor).String()
}

// Defs iterates over all definitions ("$defs" or "definitions")
func (c Core) Defs() iter.Seq2[string, Schema] {
	return nil // TODO
}

func (c Core) Def(k string) (Schema, bool) {
	if c.defsIndex == nil {
		return EmptySchema, false
	}

	idx, ok := c.defsIndex[values.MakeInternedKey(k)]
	if !ok {
		return EmptySchema, false
	}

	node := c.defs[idx]

	return schemaFromNode(c.s, node), true
}

func (c Core) HasRef() bool {
	return false
}

func (c Core) Ref() Ref {
	return Ref{}
}

func (c Core) ContentEncoding() string {
	return ""
}

func (c Core) ContentMediaType() string {
	return ""
}

func (c Core) ContentSchema() Schema {
	return Schema{}
}

func (c Core) VersionRequirements() VersionRequirements {
	return c.version
}

func (c *Core) decode(ctx *light.ParentContext, key values.InternedKey) error {
	octx, ok := ctx.X.(*schemaContext)
	if !ok {
		panic("bug")
	}

	switch key {
	case anchorKey:
		// consume node
		//octx.anchor =  ...
	case contentEncodingKey:
	case contentMediaTypeKey:
	case definitionsKey:
	case defsKey:
	case draft4IDKey:
	case dynamicAnchorKey:
	case dynamicRefKey:
	case idKey:
	case recursiveAnchorKey:
	case recursiveRefKey:
	case refKey:
	case schemaKey:
	default:
		panic("bug")
	}
	return nil // TODO
}
