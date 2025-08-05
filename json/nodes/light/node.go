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
// Use [VerbatimNode] to cover use cases hat require that the original JSON can be reconstructed verbatim.
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

// Value of a node of kind nodes.KindScalar.
func (n Node) Value(s stores.Store) (values.Value, bool) {
	switch n.kind {
	case nodes.KindScalar:
		return s.Get(n.value), true
	case nodes.KindObject, nodes.KindArray:
		fallthrough
	default:
		return values.NullValue, false
	}
}

// Handle of the alue of a node of kind nodes.KindScalar.
func (n Node) Handle() (stores.Handle, bool) {
	switch n.kind {
	case nodes.KindScalar:
		return n.value, true
	case nodes.KindObject, nodes.KindArray:
		fallthrough
	default:
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

func (n Node) IsNull(_ stores.Store) bool {
	if n.kind != nodes.KindScalar {
		return false
	}

	return n.value == stores.HandleZero
}

// AtKey returns the [Node] held under a key in an object, or false if not found.
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

// Key of the current node, if part of an object
func (n Node) Key() string {
	return n.key.String()
}

// Pairs return all (key,Node) pairs inside an object.
func (n Node) Pairs() iter.Seq2[values.InternedKey, Node] {
	if n.kind != nodes.KindObject {
		return nil
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
func (n Node) Elems() iter.Seq[Node] {
	if n.kind != nodes.KindArray {
		return nil
	}

	return func(yield func(Node) bool) {
		for _, node := range n.children {
			if !yield(node) {
				return
			}
		}
	}
}

func (n Node) IndexedElems() iter.Seq2[int, Node] {
	if n.kind != nodes.KindArray {
		return nil
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
func (n *Node) Decode(ctx *ParentContext) {
	*n = nullNode

	defer func() {
		// capture the full context of any error if the lexer supports that
		if err := ctx.L.Err(); err != nil {
			if contextErrorer, ok := ctx.L.(interface{ ErrInContext() *codes.ErrContext }); ok {
				ctx.C = contextErrorer.ErrInContext()
				ctx.C.Err = errors.Join(err, nodecodes.ErrNode)

				return
			}

			// otherwise, keep minimal context: error and offset
			ctx.C = &codes.ErrContext{
				Err:    errors.Join(err, nodecodes.ErrNode),
				Offset: ctx.L.Offset(),
			}
		}
	}()

	n.decode(ctx)
}

// Encode the [Node] hierarchy to a [writers.StoreWriter].
//
// This consumes the values stored in the provided [stores.Store].
func (n Node) Encode(ctx *ParentContext) {
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

	ctx := &ParentContext{
		S: s,
		W: jw,
	}
	writer.RedeemUnbuffered(jw)
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

		n.decodeToken(ctx, tok)
	}
}

func (n *Node) decodeToken(ctx *ParentContext, tok token.T) {
	l := ctx.L
	s := ctx.S

	if !l.Ok() {
		// short-circuit
		return
	}

	if ctx.DO.NodeHook != nil {
		// hook: callback before a node is processed
		skip, err := ctx.DO.NodeHook(ctx, l, tok)
		if err != nil {
			l.SetErr(err)
			return
		}

		if skip {
			return
		}
	}

	// we want an object, an array or a scalar value
	switch {
	case tok.IsStartObject():
		n.keysIndex = make(map[values.InternedKey]int)
		n.children = make([]Node, 0)
		n.value = s.PutNull()
		n.kind = nodes.KindObject
		n.ctx.offset = l.Offset()

		for key, value := range n.decodeObject(ctx) {
			if !l.Ok() {
				return
			}

			// check unique key: if the key exists, replace the previous value
			// option: enforce unique keys
			index, keyExists := n.keysIndex[key]
			if keyExists {
				// by default, duplicate keys
				if !ctx.DO.tolerateDuplKey {
					return
				}

				n.children[index] = value

				continue
			}

			n.children = append(n.children, value)
			n.keysIndex[key] = len(n.children) - 1
		}

		return

	case tok.IsStartArray():
		n.keysIndex = nil
		n.children = make([]Node, 0)
		n.value = s.PutNull()
		n.kind = nodes.KindArray
		n.ctx.offset = l.Offset()

		for elem := range n.decodeArray(ctx) {
			if !l.Ok() {
				return
			}

			n.children = append(n.children, elem)
		}

		return

	case tok.IsNull():
		n.keysIndex = nil
		n.children = nil
		n.kind = nodes.KindNull
		n.value = s.PutNull()
		n.ctx.offset = l.Offset()

		return

	case tok.IsBool():
		n.keysIndex = nil
		n.children = nil
		n.kind = nodes.KindScalar
		n.value = s.PutBool(tok.Bool())
		n.ctx.offset = l.Offset()

		return

	case tok.IsScalar():
		n.keysIndex = nil
		n.children = nil
		n.kind = nodes.KindScalar
		n.value = s.PutToken(tok)
		n.ctx.offset = l.Offset()

		return

	case tok.IsEOF():
		// TODO: for VerbatimNode, remember that we may have trailing blanks before EOF
		return

	default:
		// wrong token
		l.SetErr(codes.ErrInvalidToken)
		return
	}
}

func (n *Node) decodeObject(ctx *ParentContext) iter.Seq2[values.InternedKey, Node] {
	l := ctx.L

	if !l.Ok() {
		// short-circuit
		return nil
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

			tok = l.NextToken() // skip the colon separator following the key
			if !tok.IsColon() {
				l.SetErr(codes.ErrKeyColon)
				return
			}

			if ctx.DO.BeforeKey != nil {
				// hook: callback before key is processed
				skip, err := ctx.DO.BeforeKey(ctx, l, key)
				if err != nil {
					l.SetErr(err)
					return
				}
				if skip {
					continue
				}
			}

			tok = l.NextToken() // decode the next value
			if !l.Ok() {
				return
			}

			var value Node
			value.decodeToken(ctx, tok)
			if !l.Ok() {
				return
			}

			value.key = key

			if ctx.DO.AfterKey != nil {
				// hook: callback after array element
				skip, err := ctx.DO.AfterKey(ctx, l, key, value)
				if err != nil {
					l.SetErr(err)
					return
				}
				if skip {
					continue
				}
			}

			if !yield(key, value) {
				return
			}

			separator := l.NextToken()
			if !l.Ok() {
				return
			}

			if separator.IsComma() {
				continue
			}

			if separator.IsEndObject() {
				return
			}

			l.SetErr(codes.ErrInvalidToken)

			return
		}
	}
}

func (n *Node) decodeArray(ctx *ParentContext) iter.Seq[Node] {
	l := ctx.L

	if !l.Ok() {
		// short-circuit
		return nil
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
				// empty object
				return
			}

			if ctx.DO.BeforeElem != nil {
				// hook: callback before array element
				skip, err := ctx.DO.BeforeElem(ctx, l, tok)
				if err != nil {
					l.SetErr(err)
					return
				}
				if skip {
					continue
				}
			}

			ctx.P = addElemToPath(pth, ctx.P, idx)
			idx++

			var elem Node
			elem.decodeToken(ctx, tok) // decode next value
			if !l.Ok() {
				return
			}

			if ctx.DO.AfterElem != nil {
				// hook: callback after array element
				skip, err := ctx.DO.AfterElem(ctx, l, elem)
				if err != nil {
					l.SetErr(err)
					return
				}
				if skip {
					continue
				}
			}

			if !yield(elem) {
				return
			}

			separator := l.NextToken()
			if !l.Ok() {
				return
			}

			if separator.IsComma() {
				continue
			}

			if separator.IsEndArray() {
				return
			}

			l.SetErr(codes.ErrMissingComma)
			return
		}
	}
}

func (n Node) encode(ctx *ParentContext) {
	w := ctx.W
	s := ctx.S

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

		w.Key(n.children[0].key)
		n.children[0].encode(ctx)

		for _, value := range n.children[1:] {
			w.Comma()
			w.Key(value.key)
			value.encode(ctx)
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

		for _, elem := range n.children[1:] {
			w.Comma()
			elem.encode(ctx)
		}

		w.EndArray()

		return

	case nodes.KindNull:
		w.Null()

	case nodes.KindScalar:
		// short-circuit with s.Write(n.value) (no need to allocate memory to keep the value)
		s.WriteTo(w, n.value)

	default:
		w.SetErr(fmt.Errorf("invalid node: %v: %w", n.kind, nodecodes.ErrNode))

		return
	}
}
