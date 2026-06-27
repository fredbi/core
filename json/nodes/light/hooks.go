package light

import (
	"github.com/fredbi/core/json/lexers"
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores/values"
)

// hook functions to customize how the node is decoded.
//
// Every hook returns (skip, err):
//
//   - a non-nil err aborts decoding; it is routed through the lexer's error channel and carries the
//     JSON Pointer path of the current node (see [ParentContext]).
//   - skip drops the current node from the result. The decoder still consumes the node's tokens so the
//     parse stays in sync — including draining a skipped object or array subtree — but no [Node] is
//     built or yielded for it. A skip is silent: no nested hook fires for a drained subtree.
//
// "Before" hooks fire before the value is decoded (so a skip avoids materializing it at all); "After"
// hooks fire once the value is fully decoded (a skip there discards an already-built node).
type (
	// HookKeyFunc is a callback executed on every object key, before its value is decoded.
	HookKeyFunc func(ctx *ParentContext, l lexers.Lexer, key values.InternedKey) (skip bool, err error)
	// HookKeyNodeFunc is a callback executed after an object member (key + value) has been decoded.
	HookKeyNodeFunc func(ctx *ParentContext, l lexers.Lexer, key values.InternedKey, n Node) (skip bool, err error)
	// HookElemFunc is a callback executed after an array element has been decoded.
	HookElemFunc func(ctx *ParentContext, l lexers.Lexer, elem Node) (skip bool, err error)
	// HookTokenFunc is a callback executed on a value's opening token, before it is decoded.
	HookTokenFunc func(ctx *ParentContext, l lexers.Lexer, tok token.T) (skip bool, err error)
)

// type HookFunc func(Node) (Node, error)

type decodeHooks struct {
	NodeHook   HookTokenFunc
	BeforeKey  HookKeyFunc
	AfterKey   HookKeyNodeFunc
	BeforeElem HookTokenFunc
	AfterElem  HookElemFunc
	// AfterValue HookFunc
}
