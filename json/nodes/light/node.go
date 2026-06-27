package light

import (
	"bytes"
	"errors"
	"fmt"
	"iter"

	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/nodes"
	nodecodes "github.com/fredbi/core/json/nodes/error-codes"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/stores/values"
	writer "github.com/fredbi/core/json/writers/default-writer"
)

var nullNode = Node{} //nolint:gochecknoglobals

// Node in a JSON document.
//
// A node can be either null, an object, an array or some JSON scalar value (string, bool or number).
//
// Object keys and array elements may be retrieved unitarily or walked over using iterators.
//
// Keys and array elements are reproduced in the same order they were created, e.g. from the tokens produced by a JSON lexer.
//
// [Node.Decode] and [Node.Encode] maintain the order of object keys.
// [Node] does not guarantee to be verbatim: non significant space (indentations, new lines) or escaped unicode sequences are not kept verbatim.
// Use [VerbatimNode] to cover use cases that require the original JSON to be reconstructed verbatim.
//
// Duplicate keys in an object trigger an error by default (option available to tolerate duplicates).
//
// In this "light" version of the [Node], values are [stores.Handle] references to an external [stores.Store],
// injected by the caller.
//
// This makes the [Node] a very compact representation of JSON. However the interface is more complex than the interface of a JSON document.
type Node struct {
	keysIndex map[values.InternedKey]int
	key       values.InternedKey
	children  []Node
	value     stores.Handle
	kind      nodes.Kind
	ctx       Context
}

// Value of a leaf node — a scalar or a null.
//
// A JSON null is a defined value: it returns [values.NullValue] with true. The boolean is false for
// container nodes (object, array), which have no leaf value, and for the zero handle — which a
// correctly-built node never carries (guards live in the [Builder] and the decoder), so this only
// trips on a not-found sentinel from a failed [Node.AtKey]/[Node.Elem] lookup.
func (n Node) Value(s stores.Store) (values.Value, bool) {
	switch n.kind {
	case nodes.KindScalar, nodes.KindNull:
		if n.value.IsZero() {
			return values.UndefinedValue, false
		}

		return s.Get(n.value), true
	default: // object, array
		return values.UndefinedValue, false
	}
}

// Handle of a leaf node's value — a scalar or a null.
//
// As with [Node.Value], the boolean is false for container nodes and for the zero handle (a not-found
// sentinel). A null node returns its dedicated, non-zero null handle.
func (n Node) Handle() (stores.Handle, bool) {
	switch n.kind {
	case nodes.KindScalar, nodes.KindNull:
		if n.value.IsZero() {
			return stores.HandleZero, false
		}

		return n.value, true
	default: // object, array
		return stores.HandleZero, false
	}
}

func (n Node) Context() Context {
	return n.ctx
}

func (n Node) Kind() nodes.Kind {
	return n.kind
}

func (n Node) IsObject() bool {
	return n.kind == nodes.KindObject
}

func (n Node) IsArray() bool {
	return n.kind == nodes.KindArray
}

func (n Node) IsString(s stores.Store) bool {
	if n.kind != nodes.KindScalar {
		return false
	}
	v := s.Get(n.value)

	return v.Kind() == token.String
}

func (n Node) IsNumber(s stores.Store) bool {
	if n.kind != nodes.KindScalar {
		return false
	}
	v := s.Get(n.value)

	return v.Kind() == token.Number
}

func (n Node) IsBool(s stores.Store) bool {
	if n.kind != nodes.KindScalar {
		return false
	}
	v := s.Get(n.value)

	return v.Kind() == token.Boolean
}

// IsNull reports whether the node is a JSON null.
//
// Unlike the scalar-subtype predicates it needs no store: null is a dedicated [nodes.KindNull] node
// holding a proper, non-zero null handle. The zero handle is "no value"/absence, never a null, so the
// two are not conflated.
func (n Node) IsNull() bool {
	return n.kind == nodes.KindNull
}

// AtKey returns the [Node] held under a key in an object, or false if not found.
//
// When the second result is false (key absent, or n is not an object) the returned [Node] is the zero
// node, which is indistinguishable from a JSON null (KindNull is the zero kind). Always test the bool —
// do not infer presence from the node's kind.
func (n Node) AtKey(k string) (Node, bool) {
	return n.AtInternedKey(values.MakeInternedKey(k))
}

func (n Node) AtInternedKey(k values.InternedKey) (Node, bool) {
	if n.kind != nodes.KindObject {
		return nullNode, false
	}

	index, found := n.keysIndex[k]

	if !found {
		return nullNode, false
	}

	return n.children[index], true
}

// KeyIndex returns the index of a key, or false if not found.
func (n Node) KeyIndex(k string) (int, bool) {
	if n.kind != nodes.KindObject {
		return 0, false
	}

	index, found := n.keysIndex[values.MakeInternedKey(k)]

	return index, found
}

// Elem returns the i-th element of an array.
func (n Node) Elem(i int) (Node, bool) {
	if n.kind != nodes.KindArray {
		return nullNode, false
	}
	if i >= len(n.children) || i < 0 {
		return nullNode, false
	}

	return n.children[i], true
}

// Key of the node, and whether it has one (i.e. the node is an object member).
//
// Array elements, the document root, and not-found sentinel nodes have no key and return ("", false).
// This distinguishes a missing key from a legitimately empty-string key.
func (n Node) Key() (string, bool) {
	if n.key == (values.InternedKey{}) {
		return "", false
	}

	return n.key.String(), true
}

// Pairs return all (key,Node) pairs inside an object.
//
// On a node that is not an object it yields nothing (ranging it is safe, never panics).
func (n Node) Pairs() iter.Seq2[values.InternedKey, Node] {
	if n.kind != nodes.KindObject {
		// empty (non-nil) iterator: ranging a nil iter.Seq2 panics.
		return func(func(values.InternedKey, Node) bool) {}
	}

	return func(yield func(values.InternedKey, Node) bool) {
		for _, pair := range n.children {
			if !yield(pair.key, pair) {
				return
			}
		}
	}
}

// Elems returns all elements in an array.
//
// On a node that is not an array it yields nothing (ranging it is safe, never panics).
func (n Node) Elems() iter.Seq[Node] {
	if n.kind != nodes.KindArray {
		// empty (non-nil) iterator: ranging a nil iter.Seq panics.
		return func(func(Node) bool) {}
	}

	return func(yield func(Node) bool) {
		for _, node := range n.children {
			if !yield(node) {
				return
			}
		}
	}
}

// IndexedElems returns all elements in an array together with their index.
//
// On a node that is not an array it yields nothing (ranging it is safe, never panics).
func (n Node) IndexedElems() iter.Seq2[int, Node] {
	if n.kind != nodes.KindArray {
		// empty (non-nil) iterator: ranging a nil iter.Seq2 panics.
		return func(func(int, Node) bool) {}
	}

	return func(yield func(int, Node) bool) {
		for i, node := range n.children {
			if !yield(i, node) {
				return
			}
		}
	}
}

func (n Node) Len() int {
	if n.kind != nodes.KindArray && n.kind != nodes.KindObject {
		return 0
	}

	return len(n.children)
}

// Decode the hierarchy of nodes from the input provider by a [lexers.Lexer].
//
// Decode stores scalar values in the [stores.Store] provided in the [ParentContext].
// Decode reads a single JSON value from the lexer in ctx into n, replacing any previous content.
//
// It decodes exactly one top-level value and relies on the injected lexer to enforce JSON grammar:
// trailing data or a second top-level value surfaces as a lexer error (the first value is not silently
// overwritten), and empty or whitespace-only input is reported as an error. On any failure, ctx.C
// carries the error context including the JSON Pointer path of the offending node (see [ParentContext]).
func (n *Node) Decode(ctx *ParentContext) {
	*n = nullNode

	defer func() {
		// capture the full context of any error if the lexer supports that
		if err := ctx.L.Err(); err != nil {
			// On error the per-level path truncations in decodeObject/decodeArray are skipped
			// (they only run while the lexer is Ok), so ctx.P still points at the failing node.
			path := ctx.P.String()

			if contextErrorer, ok := ctx.L.(interface{ ErrInContext() *codes.ErrContext }); ok {
				ctx.C = contextErrorer.ErrInContext()
				ctx.C.Err = errors.Join(err, nodecodes.ErrNode)
				ctx.C.Path = path

				return
			}

			// otherwise, keep minimal context: error, offset and path
			ctx.C = &codes.ErrContext{
				Err:    errors.Join(err, nodecodes.ErrNode),
				Offset: ctx.L.Offset(),
				Path:   path,
			}
		}
	}()

	n.decode(ctx)
}

// Encode the [Node] hierarchy to a [writers.StoreWriter].
//
// This consumes the values stored in the provided [stores.Store].
func (n Node) Encode(ctx *ParentContext) {
	if ctx.W == nil {
		return
	}

	defer func() {
		if err := ctx.W.Err(); err != nil {
			ctx.C = &codes.ErrContext{
				Err:    errors.Join(err, nodecodes.ErrNode),
				Offset: uint64(ctx.W.Size()), //nolint:gosec // Size() is always positive.
			}
		}
	}()

	n.encode(ctx)
}

// Dump is intended to be used for debug or inspection purpose.
//
// It dumps the content of the node as a JSON string.
func (n Node) Dump(s stores.Store) string {
	var w bytes.Buffer
	jw := writer.BorrowUnbuffered(&w)
	defer writer.RedeemUnbuffered(jw) // redeem only after Encode is done writing into jw

	ctx := &ParentContext{
		S: s,
		W: jw,
	}
	n.Encode(ctx)

	return w.String()
}

func (n *Node) decode(ctx *ParentContext) {
	l := ctx.L

	if !l.Ok() {
		// short-circuit
		return
	}

	for {
		tok := l.NextToken()
		if !l.Ok() {
			return
		}

		if tok.IsEOF() {
			return
		}

		// the root value has no key and sits at depth 0.
		if !n.decodeToken(ctx, values.InternedKey{}, tok) {
			// the root was skipped (a hook returned Skip): leave an empty document.
			*n = nullNode
		}

		if ctx.stopped {
			return
		}
	}
}

// putValue assigns the store handle h to this node.
//
// The store presents an error-free interface: on failure it returns the zero [stores.Handle] rather
// than an error. This guards against that — a zero handle means the store rejected the value, which is
// reported through the lexer's error channel, and the caller must stop (returns false).
func (n *Node) putValue(ctx *ParentContext, h stores.Handle) bool {
	if h.IsZero() {
		ctx.L.SetErr(fmt.Errorf("store returned a zero handle for a stored value: %w", nodecodes.ErrNode))

		return false
	}

	n.value = h

	return true
}

// drainValue consumes the remaining tokens of a value whose opening token tok has already been read,
// leaving the lexer positioned just after that value.
//
// It is how a hook "skip" discards a value without materializing it as a [Node]: a skipped scalar is
// already fully read (no-op), while a skipped object or array must have its body drained so the parse
// stays in sync. Separators are elided by the semantic lexer, so balancing the start/end container
// tokens is enough. No hooks fire for the drained tokens — a skip is silent.
func drainValue(ctx *ParentContext, tok token.T) {
	if !tok.IsStartObject() && !tok.IsStartArray() {
		// scalar (or a stray end token handled by the caller): the token is the whole value.
		return
	}

	l := ctx.L
	for depth := 1; depth > 0; {
		t := l.NextToken()
		if !l.Ok() {
			return
		}

		switch {
		case t.IsStartObject(), t.IsStartArray():
			depth++
		case t.IsEndObject(), t.IsEndArray():
			depth--
		case t.IsEOF():
			// the value is unterminated: the lexer's grammar check normally catches this first,
			// but guard against an early EOF while draining.
			l.SetErr(codes.ErrInvalidToken)

			return
		}
	}
}

// decodeToken decodes the value whose opening token is tok into n.
//
// key is the value's object key when it is a member (the zero key otherwise). It fires the OnEnter hook
// before decoding and the OnExit hook once the value is fully decoded, and reports whether a node was
// produced: false when a hook skipped the value (drained on enter, dropped on exit) or when decoding
// failed (check the lexer's error state). On [Stop] it sets ctx.stopped and unwinds without an error.
func (n *Node) decodeToken(ctx *ParentContext, key values.InternedKey, tok token.T) (produced bool) {
	l := ctx.L
	s := ctx.S

	if !l.Ok() || ctx.stopped {
		// short-circuit
		return false
	}

	// OnEnter: fires before the value is decoded, with its opening token.
	if hook := ctx.DO.OnEnter; hook != nil {
		action, err := hook(ctx, l, HookEvent{Kind: kindOfToken(tok), Key: key, Token: tok, Depth: len(ctx.P)})
		switch {
		case err != nil:
			l.SetErr(err)
			return false
		case action == Skip:
			// discard without materializing; a composite must be drained to stay in sync.
			drainValue(ctx, tok)
			return false
		case action == Stop:
			ctx.stopped = true
			return false
		}
	}

	n.key = key

	// we want an object, an array or a scalar value
	switch {
	case tok.IsStartObject():
		n.keysIndex = make(map[values.InternedKey]int)
		n.children = make([]Node, 0)
		if !n.putValue(ctx, s.PutNull()) {
			return false
		}
		n.kind = nodes.KindObject
		n.ctx.offset = l.Offset()

		for k, value := range n.decodeObject(ctx) {
			if !l.Ok() {
				return false
			}

			// duplicate key: by default this is an error; when tolerated, the last value wins.
			index, keyExists := n.keysIndex[k]
			if keyExists {
				if !ctx.DO.tolerateDuplKey {
					// ctx.P still points at the duplicate key (the iterator is suspended at its yield
					// and skips its truncation on error), so the reported path pinpoints the offender.
					l.SetErr(fmt.Errorf("%q: %w", k.String(), nodecodes.ErrDuplicateKey))

					return false
				}

				n.children[index] = value

				continue
			}

			n.children = append(n.children, value)
			n.keysIndex[k] = len(n.children) - 1
		}

		if !l.Ok() {
			return false
		}

	case tok.IsStartArray():
		n.keysIndex = nil
		n.children = make([]Node, 0)
		if !n.putValue(ctx, s.PutNull()) {
			return false
		}
		n.kind = nodes.KindArray
		n.ctx.offset = l.Offset()

		for elem := range n.decodeArray(ctx) {
			if !l.Ok() {
				return false
			}

			n.children = append(n.children, elem)
		}

		if !l.Ok() {
			return false
		}

	case tok.IsNull():
		n.keysIndex = nil
		n.children = nil
		n.kind = nodes.KindNull
		if !n.putValue(ctx, s.PutNull()) {
			return false
		}
		n.ctx.offset = l.Offset()

	case tok.IsBool():
		n.keysIndex = nil
		n.children = nil
		n.kind = nodes.KindScalar
		if !n.putValue(ctx, s.PutBool(tok.Bool())) {
			return false
		}
		n.ctx.offset = l.Offset()

	case tok.IsScalar():
		n.keysIndex = nil
		n.children = nil
		n.kind = nodes.KindScalar
		if !n.putValue(ctx, s.PutToken(tok)) {
			return false
		}
		n.ctx.offset = l.Offset()

	case tok.IsEOF():
		// EOF where a value is expected: top-level decode() filters EOF before calling decodeToken, so
		// reaching here means an unterminated container. The lexer's grammar check normally flags this;
		// guard so a value position never silently yields an empty node.
		l.SetErr(codes.ErrInvalidToken)
		return false

	default:
		// wrong token
		l.SetErr(codes.ErrInvalidToken)
		return false
	}

	// a descendant requested Stop during the build: keep this partial node, do not fire OnExit.
	if ctx.stopped {
		return true
	}

	// OnExit: fires once the value is fully decoded, with the assembled node. For a container this is
	// the "finished" event.
	if hook := ctx.DO.OnExit; hook != nil {
		action, err := hook(ctx, l, HookEvent{Kind: n.kind, Key: key, Node: *n, Depth: len(ctx.P)})
		switch {
		case err != nil:
			l.SetErr(err)
			return false
		case action == Skip:
			return false
		case action == Stop:
			ctx.stopped = true
			return true
		}
	}

	return true
}

func (n *Node) decodeObject(ctx *ParentContext) iter.Seq2[values.InternedKey, Node] {
	l := ctx.L

	if !l.Ok() {
		// short-circuit with an empty (non-nil) iterator: ranging a nil iter.Seq2 panics.
		return func(func(values.InternedKey, Node) bool) {}
	}

	pth := ctx.P
	lpth := len(ctx.P)

	return func(yield func(values.InternedKey, Node) bool) {
		defer func() {
			if l.Ok() {
				ctx.P = ctx.P[:lpth]
			}
		}()

		for {
			tok := l.NextToken()
			if !l.Ok() {
				return
			}

			if tok.IsEndObject() {
				// empty object
				return
			}

			if !tok.IsKey() {
				l.SetErr(codes.ErrMissingKey)
				return
			}

			key := values.MakeInternedKey(string(tok.Value()))
			ctx.P = addKeyToPath(pth, ctx.P, key)

			// note: the key/value ":" separator is elided by the semantic lexer (it validates the
			// grammar but does not emit "," and ":"), so the next token is the value itself.
			vtok := l.NextToken()
			if !l.Ok() {
				return
			}

			// decodeToken fires OnEnter/OnExit for the member value (carrying the key) and sets value.key.
			var value Node
			produced := value.decodeToken(ctx, key, vtok)
			if !l.Ok() {
				return
			}

			if produced && !yield(key, value) {
				return
			}

			if ctx.stopped {
				return
			}

			// the "," member separator is elided: loop back to read the next key or the closing "}".
		}
	}
}

func (n *Node) decodeArray(ctx *ParentContext) iter.Seq[Node] {
	l := ctx.L

	if !l.Ok() {
		// short-circuit with an empty (non-nil) iterator: ranging a nil iter.Seq panics.
		return func(func(Node) bool) {}
	}

	pth := ctx.P
	lpth := len(ctx.P)

	return func(yield func(Node) bool) {
		defer func() {
			if l.Ok() {
				ctx.P = ctx.P[:lpth]
			}
		}()

		var idx int

		for {
			tok := l.NextToken()
			if !l.Ok() {
				return
			}

			if tok.IsEndArray() {
				// empty array
				return
			}

			ctx.P = addElemToPath(pth, ctx.P, idx)
			idx++

			// decodeToken fires OnEnter/OnExit for the element value (array elements have no key).
			var elem Node
			produced := elem.decodeToken(ctx, values.InternedKey{}, tok)
			if !l.Ok() {
				return
			}

			if produced && !yield(elem) {
				return
			}

			if ctx.stopped {
				return
			}

			// the "," element separator is elided: loop back to read the next element or the closing "]".
		}
	}
}

// encode walks the node hierarchy and drives the [writers.StoreWriter].
//
// It uses a pointer receiver to avoid copying the Node on every recursive call; it never mutates the
// node, so re-encoding the same hierarchy is safe.
//
// TODO(fred): recursion is bounded only by the runtime stack. Decode-time nesting is already capped by
// the lexer's max-depth option (the caller injects the lexer), so a decoded hierarchy is safe;
// programmatically built trees are the builder caller's responsibility. Revisit if a hard encode-side
// depth guard is wanted.
func (n *Node) encode(ctx *ParentContext) {
	w := ctx.W
	if w == nil {
		return
	}

	if !w.Ok() {
		return
	}

	switch n.kind {
	case nodes.KindObject:
		w.StartObject()
		if !w.Ok() {
			return
		}

		if len(n.children) == 0 {
			w.EndObject()

			return
		}

		// the builder and decoder guarantee every child of an object carries a valid key, so the
		// encoder writes keys without re-validating them here.
		w.Key(n.children[0].key)
		n.children[0].encode(ctx)

		for i := 1; i < len(n.children); i++ {
			if !w.Ok() {
				return
			}
			w.Comma()
			w.Key(n.children[i].key)
			n.children[i].encode(ctx)
		}

		if !w.Ok() {
			return
		}
		w.EndObject()

		return

	case nodes.KindArray:
		w.StartArray()
		if !w.Ok() {
			return
		}

		if len(n.children) == 0 {
			w.EndArray()

			return
		}

		n.children[0].encode(ctx)

		for i := 1; i < len(n.children); i++ {
			if !w.Ok() {
				return
			}
			w.Comma()
			n.children[i].encode(ctx)
		}

		if !w.Ok() {
			return
		}
		w.EndArray()

		return

	case nodes.KindNull:
		w.Null()

	case nodes.KindScalar:
		// A scalar must point to a real value handle. HandleZero means "no value" (an absent or
		// corrupted handle), which is distinct from a JSON null (KindNull holds a proper null handle).
		// WriteTo(HandleZero) would silently emit nothing and produce invalid JSON, so flag it instead.
		if n.value.IsZero() {
			w.SetErr(fmt.Errorf(
				"scalar node has a zero (absent) value handle: %w",
				nodecodes.ErrNode,
			))

			return
		}

		if ctx.S == nil {
			w.SetErr(fmt.Errorf("nil store: cannot resolve scalar value: %w", nodecodes.ErrNode))

			return
		}

		// short-circuit with s.Write(n.value) (no need to allocate memory to keep the value)
		ctx.S.WriteTo(w, n.value)

	default:
		w.SetErr(fmt.Errorf("invalid node: %v: %w", n.kind, nodecodes.ErrNode))

		return
	}
}
