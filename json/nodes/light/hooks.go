package light

import (
	"github.com/fredbi/core/json/lexers"
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/nodes"
	"github.com/fredbi/core/json/stores/values"
)

// Action tells the decoder what to do after a [Hook] has observed an event.
type Action uint8

const (
	// Continue decodes the value normally (the default, zero value).
	Continue Action = iota

	// Skip discards the value from the result. On [decodeHooks.OnEnter] the value is drained from the
	// lexer without being materialized (a composite subtree is consumed but not built); on
	// [decodeHooks.OnExit] the already-built node is dropped. The parse stays in sync either way, and a
	// skip is silent — no nested hook fires for a drained subtree.
	Skip

	// Stop ends decoding immediately, keeping whatever has been built so far, without an error. On
	// OnExit the value that triggered Stop is kept; on OnEnter it is not (it was never decoded).
	Stop
)

// HookEvent is the payload handed to a [Hook]. Which fields are populated depends on the moment:
//
//   - on enter (before a value is decoded): Kind, Key (if the value is an object member), Token, Depth.
//   - on exit (after a value is fully decoded): Kind, Key (if a member), Node, Depth.
//
// The single payload keeps the callback surface to one signature while leaving room to grow (new
// fields/events never add new signatures).
type HookEvent struct {
	// Kind of the value in play (derived from Token on enter, from the node on exit).
	Kind nodes.Kind
	// Key of the value when it is an object member; the zero key otherwise (array element, root).
	Key values.InternedKey
	// Token is the value's opening token. Set on enter, zero on exit.
	Token token.T
	// Node is the fully-decoded value. Set on exit, zero on enter.
	Node Node
	// Depth is the nesting depth of the value: 0 at the document root, 1 for a top-level member, etc.
	Depth int
}

// HasKey reports whether the value is an object member (i.e. [HookEvent.Key] is set). It is false for
// array elements and for the document root.
func (ev HookEvent) HasKey() bool {
	return ev.Key != values.InternedKey{}
}

// Hook is the single callback signature used for every decode event.
//
// It returns an [Action] controlling the decode flow, or a non-nil error to abort decoding (the error
// is routed through the lexer's error channel and carries the JSON Pointer path of the value — see
// [ParentContext]). A returned error takes precedence over the Action.
type Hook func(ctx *ParentContext, l lexers.Lexer, ev HookEvent) (Action, error)

// decodeHooks are the general-purpose callbacks fired while decoding a node hierarchy.
//
// Every value, at every depth including the document root, fires OnEnter before it is decoded and (if
// not skipped/stopped) OnExit once it is fully decoded. OnExit on a container is the "object/array
// finished" event — e.g. to check required keys, detect a "$ref", or enforce array constraints.
type decodeHooks struct {
	// OnEnter fires before a value is decoded, with its opening Token (and Key if a member).
	OnEnter Hook
	// OnExit fires after a value is fully decoded, with its Node (and Key if a member). For a container
	// this is the moment it is complete.
	OnExit Hook
}

// kindOfToken maps a value's opening token to its [nodes.Kind].
func kindOfToken(tok token.T) nodes.Kind {
	switch {
	case tok.IsStartObject():
		return nodes.KindObject
	case tok.IsStartArray():
		return nodes.KindArray
	case tok.IsNull():
		return nodes.KindNull
	default:
		return nodes.KindScalar
	}
}
