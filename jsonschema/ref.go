package jsonschema

import (
	"context"
	"errors"
)

// Ref knows how to resolve a JSON reference
// $id, $ref, $dynamicRef, $anchor, $dynamicAnchor
//
// $ref semantics depends on the version of the schema:
//
//   - for JSON schema draft 4 to 7: if a $ref is present, all other keys are ignored.
//   - from JSON schema draft 2019 onwards, $ref is evaluated but sibling keys remain
//
// TODO: check draft 6 assertion "it is now possible to describe instance properties named $ref"
type Ref struct {
	uri    string // TODO: URI
	cached *Schema
}

type ResolverContext struct {
	cache     map[string]Schema
	basePath  string
	baseStack []string // or container.List ?
}

func defaultResolverContext() *ResolverContext {
	return &ResolverContext{
		baseStack: []string{},
	}
}

type refCtxKey uint8

const resolverKey refCtxKey = iota + 1

// Resolve a $ref as a [Schema]
func (r *Ref) Resolve(ctx context.Context) (Schema, error) {
	if r.cached != nil {
		return *r.cached, nil
	}

	rctx, ok := ctx.Value(resolverKey).(*ResolverContext)
	if !ok {
		rctx = defaultResolverContext()
	}

	// currentBase := "" // TODO
	defer func() {
		// TODO: only if base changes
		rctx.baseStack = append(rctx.baseStack, "")
	}()

	// resolved = true

	return Schema{}, errors.ErrUnsupported
}

func (r *Ref) String() string {
	return r.uri // TODO
}

// ResolveRecurse resolves a $ref recursively until all nested $ref are exhausted
func (r Ref) ResolveRecurse(ctx context.Context) (Schema, error) {
	return Schema{}, errors.ErrUnsupported
}
