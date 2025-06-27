package jsonschema

import (
	"context"
	"errors"
)

type Ref struct {
	resolved bool
	schema   Schema
}

type ResolverContext struct {
	basePath  string
	baseStack []string
}

func defaultResolverContext() *ResolverContext {
	return ResolverContext{
		baseStack: []string{},
	}
}

type ctxKey uint8

const resolverKey ctxKey = iota + 1

func (r Ref) Resolve(ctx context.Context) (Schema, error) {
	if r.resolved {
		return r.schema, nil
	}

	rctx, ok := ctx.Value(resolverKey).(*ResolverContext)
	if !ok {
		rctx = defaultResolverContext()
	}

	currentBase := "" // TODO
	defer func() {
		// TODO: only if base changes
		rctx.baseStack = append(rctx.baseStack, "")
	}()

	// resolved = true

	return Schema{}, errors.ErrUnsupported
}

func (r *Ref) String() string {
	return "" // TODO
}

func (r Ref) ResolveRecurse(ctx context.Context) (Schema, error) {
	return Schema{}, errors.ErrUnsupported
}
