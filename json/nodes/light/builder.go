package light

import (
	"errors"
	"fmt"
	"maps"
	"math/big"
	"slices"

	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/nodes"
	nodecodes "github.com/fredbi/core/json/nodes/error-codes"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/json/types"
	"github.com/fredbi/core/swag/conv"
)

// Builder to construct a [Node] programmatically.
//
// A [Node] is immutable: cloning one (via [Builder.From]) and mutating the clone never alters the
// original. The builder achieves this with copy-on-write — see [Builder.cloneForWrite]. Cloning is
// cheap (a shallow share) and a mutation chain copies the shared slice and index at most once.
type Builder struct {
	s   stores.Store
	err error
	n   Node

	// aliased reports that n.children and n.keysIndex may be shared with a foreign Node (seeded via
	// From) or with a snapshot already handed out by Node. While set, the next mutation must clone
	// them before writing (copy-on-write).
	aliased bool
}

// NewBuilder yields a fresh [Node] builder.
func NewBuilder(s stores.Store) *Builder {
	return &Builder{
		s: s,
	}
}

func (b Builder) Err() error {
	return b.err
}

func (b Builder) Ok() bool {
	return b.err == nil
}

func (b *Builder) SetErr(err error) {
	b.err = err
}

// setValue assigns the store handle h to the node being built.
//
// The store presents an error-free interface: on failure it returns the zero [stores.Handle] rather
// than an error. This guards against that — a zero handle means the store rejected the value, which is
// recorded as a builder error so the chain short-circuits on the next [Builder.Ok] check.
func (b *Builder) setValue(h stores.Handle) {
	if h.IsZero() {
		b.err = fmt.Errorf("store returned a zero handle for a stored value: %w", nodecodes.ErrBuilder)

		return
	}

	b.n.value = h
}

func (b *Builder) Reset() {
	b.err = nil
	b.n = nullNode
	b.aliased = false
}

func (b *Builder) WithStore(s stores.Store) *Builder {
	b.s = s

	return b
}

// Node returns the [Node] produced by the [Builder].
//
// If a build error has occurred, it returns the empty [Node], which corresponds to JSON null.
func (b *Builder) Node() Node {
	if !b.Ok() {
		return nullNode
	}

	// the returned Node shares this builder's slice and index; mark aliased so the next mutation
	// copies-on-write and leaves the handed-out snapshot unaltered.
	b.aliased = true

	return b.n
}

func (b *Builder) From(n Node) *Builder {
	// seed from a foreign Node: share its slice and index read-only (no copy). The first mutation
	// will clone-on-write, so n is never altered.
	b.n = n
	b.aliased = true

	return b
}

func (b *Builder) WithContext(c Context) *Builder {
	b.n.ctx = c

	return b
}

// Object builds a JSON object
func (b *Builder) Object() *Builder {
	if !b.Ok() {
		return b
	}

	b.n.kind = nodes.KindObject
	b.resetNode()

	return b
}

func (b *Builder) Array() *Builder {
	if !b.Ok() {
		return b
	}

	b.n.kind = nodes.KindArray
	b.resetNode()

	return b
}

// ClearChildren removes children nodes in an object or array.
func (b *Builder) ClearChildren() *Builder {
	if !b.Ok() {
		return b
	}

	if b.n.kind != nodes.KindArray && b.n.kind != nodes.KindObject {
		b.err = fmt.Errorf(
			"can't clear the children of a non-container node. Node kind is %v: %w",
			b.n.kind, nodecodes.ErrBuilder,
		)

		return b
	}

	b.resetNode()

	return b
}

// Swap two children nodes in an object or array.
func (b *Builder) Swap(i, j int) *Builder {
	if !b.Ok() {
		return b
	}

	if b.n.kind != nodes.KindArray && b.n.kind != nodes.KindObject {
		b.err = fmt.Errorf(
			"can't swap the children of a non-container node. Node kind is %v: %w",
			b.n.kind, nodecodes.ErrBuilder,
		)

		return b
	}

	if i < 0 || j < 0 || i >= len(b.n.children) || j >= len(b.n.children) {
		b.err = fmt.Errorf(
			"can't swap out of range children. Indices (%d,%d) out of [0,%d): %w",
			i, j, len(b.n.children), nodecodes.ErrBuilder,
		)

		return b
	}

	b.cloneForWrite()

	// only objects carry a key index; arrays must not get a phantom keysIndex map.
	if b.n.kind == nodes.KindObject {
		keyi := b.n.children[i].key
		keyj := b.n.children[j].key
		b.n.keysIndex[keyi] = j
		b.n.keysIndex[keyj] = i
	}

	b.n.children[i], b.n.children[j] = b.n.children[j], b.n.children[i]

	return b
}

func (b *Builder) AppendKey(key string, value Node) *Builder {
	if !b.Ok() {
		return b
	}

	if !b.requireObject("add a key to") {
		return b
	}

	b.cloneForWrite()
	b.ensureIndex()
	value.key = values.MakeInternedKey(key)
	if b.rejectDuplicateKey(key, value.key) {
		return b
	}

	b.n.children = append(b.n.children, value)
	b.n.keysIndex[value.key] = len(b.n.children) - 1

	return b
}

func (b *Builder) PrependKey(key string, value Node) *Builder {
	if !b.Ok() {
		return b
	}

	if !b.requireObject("prepend a key into") {
		return b
	}

	b.cloneForWrite()
	b.ensureIndex()
	value.key = values.MakeInternedKey(key)
	if b.rejectDuplicateKey(key, value.key) {
		return b
	}

	b.n.children = slices.Insert(b.n.children, 0, value)

	for k := range b.n.keysIndex {
		b.n.keysIndex[k]++
	}
	b.n.keysIndex[value.key] = 0

	return b
}

func (b *Builder) InsertKey(key string, position int, value Node) *Builder {
	if !b.Ok() {
		return b
	}

	if !b.requireObject("insert a key into") {
		return b
	}

	if position <= 0 {
		return b.PrependKey(key, value)
	}

	if position >= len(b.n.children) {
		return b.AppendKey(key, value)
	}

	b.cloneForWrite()
	b.ensureIndex()
	value.key = values.MakeInternedKey(key)
	if b.rejectDuplicateKey(key, value.key) {
		return b
	}

	b.n.children = slices.Insert(b.n.children, position, value)

	for k, index := range b.n.keysIndex {
		if index >= position {
			b.n.keysIndex[k]++
		}
	}
	b.n.keysIndex[value.key] = position

	return b
}

func (b *Builder) RemoveKey(key string) *Builder {
	if !b.Ok() {
		return b
	}

	if !b.requireObject("remove a key from") {
		return b
	}

	b.ensureIndex()
	k := values.MakeInternedKey(key)
	index, ok := b.n.keysIndex[k]
	if !ok {
		// key is not present: no error (and no copy-on-write — nothing is mutated)
		return b
	}

	b.cloneForWrite()
	delete(b.n.keysIndex, k)
	b.n.children = slices.Delete(b.n.children, index, index+1)

	// every key past the removed slot has shifted left by one: re-index.
	for ki, kindex := range b.n.keysIndex {
		if kindex > index {
			b.n.keysIndex[ki] = kindex - 1
		}
	}

	return b
}

func (b *Builder) AppendElem(value Node) *Builder {
	if !b.Ok() {
		return b
	}

	if !b.requireArray("add an element to") {
		return b
	}

	b.cloneForWrite()
	b.ensureChildren()
	b.n.children = append(b.n.children, value)

	return b
}

func (b *Builder) PrependElem(value Node) *Builder {
	if !b.Ok() {
		return b
	}

	if !b.requireArray("add an element to") {
		return b
	}

	b.cloneForWrite()
	b.ensureChildren()
	b.n.children = slices.Insert(b.n.children, 0, value)

	return b
}

func (b *Builder) InsertElem(position int, value Node) *Builder {
	if !b.Ok() {
		return b
	}

	if !b.requireArray("add an element to") {
		return b
	}

	if position < 0 {
		return b.PrependElem(value)
	}

	if position >= len(b.n.children) {
		return b.AppendElem(value)
	}

	b.cloneForWrite()
	b.ensureChildren()
	b.n.children = slices.Insert(b.n.children, position, value)

	return b
}

func (b *Builder) RemoveElem(position int) *Builder {
	if !b.Ok() {
		return b
	}

	if !b.requireArray("remove an element from") {
		return b
	}
	if position >= len(b.n.children) || position < 0 {
		b.err = fmt.Errorf(
			"can't remove an out of range element. %d >= %d: %w",
			position, len(b.n.children), nodecodes.ErrBuilder,
		)

		return b
	}

	b.cloneForWrite()
	b.ensureChildren()
	b.n.children = slices.Delete(b.n.children, position, position+1)

	return b
}

func (b *Builder) AppendElems(values ...Node) *Builder {
	if !b.Ok() {
		return b
	}

	for _, value := range values {
		_ = b.AppendElem(value)
		if !b.Ok() {
			break
		}
	}

	return b
}

// StringValue builds a scalar node of type string.
func (b *Builder) StringValue(value string) *Builder {
	if !b.Ok() {
		return b
	}

	b.n.kind = nodes.KindScalar
	b.resetNode()

	b.setValue(b.s.PutValue(values.MakeStringValue(value)))

	return b
}

func (b *Builder) BytesValue(value []byte) *Builder {
	if !b.Ok() {
		return b
	}

	b.n.kind = nodes.KindScalar
	b.resetNode()

	b.setValue(b.s.PutValue(values.MakeScalarValue(token.MakeWithValue(token.String, value))))

	return b
}

func (b *Builder) BoolValue(value bool) *Builder {
	if !b.Ok() {
		return b
	}

	b.n.kind = nodes.KindScalar
	b.resetNode()

	b.setValue(b.s.PutBool(value))

	return b
}

func (b *Builder) NumberValue(value types.Number) *Builder {
	if !b.Ok() {
		return b
	}

	b.n.kind = nodes.KindScalar
	b.resetNode()

	b.setValue(b.s.PutValue(values.MakeNumberValue(value)))

	return b
}

func (b *Builder) NumericalValue(value any) *Builder {
	if !b.Ok() {
		return b
	}

	switch v := value.(type) {
	case float64:
		return buildFromFloat(b, v)
	case float32:
		return buildFromFloat(b, v)
	case int64:
		return buildFromInteger(b, v)
	case int32:
		return buildFromInteger(b, v)
	case int16:
		return buildFromInteger(b, v)
	case int8:
		return buildFromInteger(b, v)
	case int:
		return buildFromInteger(b, v)
	case uint64:
		return buildFromUinteger(b, v)
	case uint32:
		return buildFromUinteger(b, v)
	case uint16:
		return buildFromUinteger(b, v)
	case uint8:
		return buildFromUinteger(b, v)
	case uint:
		return buildFromUinteger(b, v)
	case *big.Int:
		if v == nil {
			return b
		}
		return buildFromTextAppender(b, v)
	case big.Int:
		return buildFromTextAppender(b, &v)

	case *big.Float:
		if v == nil {
			return b
		}
		return buildFromTextAppender(b, v)
	case big.Float:
		return buildFromTextAppender(b, &v)

	case *big.Rat:
		if v == nil {
			return b
		}
		f, _ := v.Float64()

		return buildFromFloat(b, f)

	case big.Rat:
		f, _ := v.Float64()

		return buildFromFloat(b, f)

	case []byte:
		if len(v) == 0 {
			return b
		}

		var bf big.Float
		if err := bf.UnmarshalText(v); err != nil {
			b.err = fmt.Errorf(
				"method NumericalValue could not convert the input %T to a JSON number: %q: %w",
				value, value, nodecodes.ErrBuilder,
			)

			return b
		}

		return buildFromTextAppender(b, &bf)

	case string:
		if v == "" {
			return b
		}

		var bf big.Float
		if err := bf.UnmarshalText([]byte(v)); err != nil {
			b.err = fmt.Errorf(
				"method NumericalValue could not convert the input %T to a JSON number: %q: %w",
				value, value, nodecodes.ErrBuilder,
			)

			return b
		}

		return buildFromTextAppender(b, &bf)

	default:
		b.err = fmt.Errorf(
			"method NumericalValue could not convert the input of type %T to a JSON number: %w",
			value, nodecodes.ErrBuilder,
		)

		return b
	}
}

// Float64Value builds a number node from a float64 value.
func (b *Builder) Float64Value(value float64) *Builder {
	if !b.Ok() {
		return b
	}

	return buildFromFloat(b, value)
}

// Float32Value builds a number node from a float32 value.
func (b *Builder) Float32Value(value float32) *Builder {
	if !b.Ok() {
		return b
	}

	return buildFromFloat(b, value)
}

// IntegerValue builds a number node from an int64 value.
func (b *Builder) IntegerValue(value int64) *Builder {
	if !b.Ok() {
		return b
	}

	return buildFromInteger(b, value)
}

func (b *Builder) UintegerValue(value uint64) *Builder {
	if !b.Ok() {
		return b
	}

	return buildFromUinteger(b, value)
}

// Null builds a node with "null".
func (b *Builder) Null() *Builder {
	if !b.Ok() {
		return b
	}

	b.n = nullNode
	b.aliased = false

	return b
}

// cloneForWrite implements copy-on-write.
//
// The first mutation after the builder was seeded from a foreign [Node] (via [Builder.From]) or after
// it handed out a snapshot (via [Builder.Node]) clones the shared slice and index, so the original is
// never altered. Clearing the aliased flag makes every subsequent mutation in the same chain a no-op
// here: a mutation chain therefore copies at most once, no matter how long it is.
func (b *Builder) cloneForWrite() {
	if !b.aliased {
		return
	}

	b.n.children = slices.Clone(b.n.children)
	if b.n.keysIndex != nil {
		b.n.keysIndex = maps.Clone(b.n.keysIndex)
	}
	b.aliased = false
}

func (b *Builder) resetNode() {
	if b.aliased {
		// the slice and index are shared read-only; don't reuse them, start fresh so the original
		// (or a handed-out snapshot) is left intact.
		b.n.children = nil
		b.n.keysIndex = nil
		b.aliased = false

		return
	}

	if b.n.children != nil {
		b.n.children = b.n.children[:0]
	}
	if b.n.keysIndex != nil {
		clear(b.n.keysIndex)
	}
}

func (b *Builder) ensureIndex() {
	if b.n.keysIndex == nil {
		b.n.keysIndex = make(map[values.InternedKey]int)
	}

	if b.n.children == nil {
		b.n.children = make([]Node, 0)
	}
}

func (b *Builder) ensureChildren() {
	if b.n.children == nil {
		b.n.children = make([]Node, 0)
	}
}

// requireObject sets a build error and returns false when the current node is not an object.
// action describes the attempted operation, e.g. "add a key to".
func (b *Builder) requireObject(action string) bool {
	if b.n.kind != nodes.KindObject {
		b.err = fmt.Errorf(
			"can't %s a non-object node. Node kind is %v: %w",
			action, b.n.kind, nodecodes.ErrBuilder,
		)

		return false
	}

	return true
}

// requireArray sets a build error and returns false when the current node is not an array.
// action describes the attempted operation, e.g. "add an element to".
func (b *Builder) requireArray(action string) bool {
	if b.n.kind != nodes.KindArray {
		b.err = fmt.Errorf(
			"can't %s a non-array node. Node kind is %v: %w",
			action, b.n.kind, nodecodes.ErrBuilder,
		)

		return false
	}

	return true
}

// rejectDuplicateKey sets a build error and returns true when ik is already present in the object.
func (b *Builder) rejectDuplicateKey(key string, ik values.InternedKey) bool {
	if _, ok := b.n.keysIndex[ik]; ok {
		b.err = fmt.Errorf(
			"key is already present in object: %q: %w",
			key, nodecodes.ErrBuilder,
		)

		return true
	}

	return false
}

func buildFromFloat[T conv.Float](b *Builder, value T) *Builder {
	b.n.kind = nodes.KindScalar
	b.resetNode()
	b.setValue(b.s.PutValue(values.MakeFloatValue(value)))

	return b
}

func buildFromInteger[T conv.Signed](b *Builder, value T) *Builder {
	b.n.kind = nodes.KindScalar
	b.resetNode()
	b.setValue(b.s.PutValue(values.MakeIntegerValue(value)))

	return b
}

func buildFromUinteger[T conv.Unsigned](b *Builder, value T) *Builder {
	b.n.kind = nodes.KindScalar
	b.resetNode()
	b.setValue(b.s.PutValue(values.MakeUintegerValue(value)))

	return b
}

func buildFromTextAppender(b *Builder, v interface{ AppendText([]byte) ([]byte, error) }) *Builder {
	const (
		sensibleNumberLength = 20
	)

	buf := make([]byte, 0, sensibleNumberLength)
	value, err := v.AppendText(buf)
	if err != nil {
		b.err = errors.Join(err, nodecodes.ErrBuilder)

		return b
	}

	b.n.kind = nodes.KindScalar
	b.resetNode()
	b.setValue(b.s.PutValue(values.MakeNumberValue(types.Number{Value: value})))

	return b
}
