package light

import (
	"fmt"
	"iter"

	"github.com/fredbi/core/json/lexers"
	codes "github.com/fredbi/core/json/lexers/error-codes"
	"github.com/fredbi/core/json/nodes"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/writers"
)

// Context holds the lexer's offset for every decoded token.
//
// Context does not apply to a [Node] built programmatically wih a [Builder].
type Context struct {
	offset uint64
}

// Offset of this node in the original JSON stream.
func (c Context) Offset() uint64 {
	return c.offset
}

var nullNode = Node{}

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
// # Duplicate keys in an object? TODO
//
// In this "light" version of the [Node], values are [stores.Handle] references to an external [stores.Store],
// injected by the caller.
//
// This makes the [Node] a very compact representation of JSON, but the interface is more complex than the interface of a JSON document.
type Node struct {
	keysIndex map[stores.InternedKey]int
	key       stores.InternedKey
	children  []Node
	value     stores.Handle
	kind      nodes.Kind
	ctx       Context
}

// ParentContext injects all the dependencies needed to operate with a [Node].
type ParentContext struct {
	S  stores.Store
	L  lexers.Lexer
	W  writers.Writer
	DO DecodeOptions
	EO EncodeOptions
	C  lexer.Context // TODO : lexer context
}

func (n Node) Value(s stores.Store) (stores.Value, bool) {
	switch n.kind {
	case nodes.KindScalar:
		return s.Get(n.value), true
	case nodes.KindObject, nodes.KindArray:
		fallthrough
	default:
		return stores.NullValue, false
	}
}

func (n Node) Context() Context {
	return n.ctx
}

func (n Node) Kind() nodes.Kind {
	return n.kind
}

// AtKey returns the [Node] held under a key in an object, or false if not found.
func (n Node) AtKey(k string) (Node, bool) {
	if n.kind != nodes.KindObject {
		return nullNode, false
	}

	index, found := n.keysIndex[stores.MakeInternedKey(k)]

	if !found {
		return nullNode, false
	}

	return n.children[index], true
}

// KeyIndex returns the index of a key, or false if not found.
func (n Node) KeyIndex(k string) (int, bool) {
	index, found := n.keysIndex[stores.MakeInternedKey(k)]

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
func (n Node) Pairs() iter.Seq2[stores.InternedKey, Node] {
	if n.kind != nodes.KindObject {
		return nil
	}

	return func(yield func(stores.InternedKey, Node) bool) {
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
	n.decode(ctx)
}

// Encode the [Node] hierarchy to a [writers.Writer].
//
// This consumes the values stored in the provided [stores.Store].
func (n Node) Encode(ctx *ParentContext) {
	n.encode(ctx)
}

func (n *Node) decode(ctx *ParentContext) {
	l := ctx.L
	s := ctx.S

	if !l.Ok() {
		// short-circuit
		return
	}

	for {
		tok := l.NextToken()
		if !l.Ok() {
			// TODO: set error and context
			return
		}
		if ctx.DO.NodeHook != nil {
			// hook: callback before a node is processed
			skip, err := ctx.DO.NodeHook(l, tok)
			if err != nil {
				l.SetErr(err)
				return
			}

			if skip {
				continue
			}
		}

		// we want an object, an array or a scalar value
		switch {
		case tok.IsStartObject():
			n.keysIndex = make(map[stores.InternedKey]int)
			n.children = make([]Node, 0)
			n.value = s.PutNull()
			n.kind = nodes.KindObject
			n.ctx.offset = l.Offset()

			for key, value := range n.decodeObject(ctx) {
				if !l.Ok() {
					// TODO: set error and context
					return
				}
				// check unique key: if the key exists, replace the previous value
				// option: enforce unique keys
				index, keyExists := n.keysIndex[key]
				if keyExists {
					if ctx.DO.uniqueKey {
						// set error TODO
						return
					}
					n.children[index] = value

					continue
				}

				n.children = append(n.children, value)
				n.keysIndex[key] = len(n.children) - 1
			}

		case tok.IsStartArray():
			n.keysIndex = nil
			n.children = make([]Node, 0)
			n.value = s.PutNull()
			n.kind = nodes.KindArray
			n.ctx.offset = l.Offset()

			for elem := range n.decodeArray(ctx) {
				if !l.Ok() {
					// TODO: set error and context
					return
				}

				n.children = append(n.children, elem)
			}

		case tok.IsNull():
			n.keysIndex = nil
			n.children = nil
			n.kind = nodes.KindNull
			n.value = s.PutNull()
			n.ctx.offset = l.Offset()

		case tok.IsBool():
			n.keysIndex = nil
			n.children = nil
			n.kind = nodes.KindScalar
			n.value = s.PutBool(tok.Bool())
			n.ctx.offset = l.Offset()

		case tok.IsScalar():
			n.keysIndex = nil
			n.children = nil
			n.kind = nodes.KindScalar
			n.value = s.PutToken(tok)
			n.ctx.offset = l.Offset()

		case tok.IsEOF():
			// TODO: for VerbatimNode, remember that we may have trailing blanks before EOF
			return

		default:
			// wrong
			l.SetErr(codes.ErrInvalidToken)
			return
		}
	}
}

func (n *Node) decodeObject(ctx *ParentContext) iter.Seq2[stores.InternedKey, Node] {
	l := ctx.L

	if !l.Ok() {
		// short-circuit
		return nil
	}

	return func(yield func(stores.InternedKey, Node) bool) {
		for {
			tok := l.NextToken()
			if !l.Ok() {
				return
			}

			if tok.IsEndObject() {
				// empty object
				return
			}

			if tok.IsKey() {
				l.SetErr(codes.ErrMissingKey)
				return
			}

			tok = l.NextToken() // skip the colon separator following the key
			if !tok.IsColon() {
				l.SetErr(codes.ErrKeyColon)
				return
			}

			key := stores.MakeInternedKey(string(tok.Value()))

			if ctx.DO.BeforeKey != nil {
				// hook: callback before key is processed
				skip, err := ctx.DO.BeforeKey(l, key)
				if err != nil {
					l.SetErr(err)
					return
				}
				if skip {
					continue
				}
			}

			var value Node
			value.decode(ctx)
			if !l.Ok() {
				return
			}
			value.key = key

			if ctx.DO.AfterKey != nil {
				// hook: callback after array element
				skip, err := ctx.DO.AfterKey(l, key, value)
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

	return func(yield func(Node) bool) {
		for {
			tok := l.NextToken()
			if !l.Ok() {
				return
			}

			if tok.IsEndArray() {
				// empty object
				return
			}

			var elem Node
			elem.decode(ctx)
			if !l.Ok() {
				return
			}

			if ctx.DO.AfterElem != nil {
				// hook: callback after array element
				skip, err := ctx.DO.AfterElem(l, elem)
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

	case nodes.KindScalar:
		w.Value(s.Get(n.value))

	case nodes.KindNull:
		w.Null()

	default:
		w.SetErr(fmt.Errorf("invalid node: %v", n.kind)) // TODO: wrap package-level sentinel error
		return
	}
}
