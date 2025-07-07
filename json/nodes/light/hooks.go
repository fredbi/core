package light

import (
	"github.com/fredbi/core/json/lexers"
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores/values"
)

type (
	// hook functions to customize how the node is decoded.
	HookKeyFunc     func(l lexers.Lexer, key values.InternedKey) (skip bool, err error)
	HookKeyNodeFunc func(l lexers.Lexer, key values.InternedKey, n Node) (skip bool, err error)
	HookElemFunc    func(l lexers.Lexer, elem Node) (skip bool, err error)
	HookTokenFunc   func(l lexers.Lexer, tok token.T) (skip bool, err error)
)

// type HookFunc func(Node) (Node, error)

type decodeHooks struct {
	NodeHook  HookTokenFunc
	BeforeKey HookKeyFunc
	AfterKey  HookKeyNodeFunc
	AfterElem HookElemFunc
	// AfterValue HookFunc
}
