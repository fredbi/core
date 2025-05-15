package light

import (
	"fmt"
	"slices"

	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/nodes"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/types"
	"github.com/fredbi/core/swag/conv"
)

// Builder to construct a [Node] programmatically.
type Builder struct {
	s   stores.Store
	err error
	n   Node
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

func (b *Builder) Reset() {
	b.err = nil
	b.n = nullNode
}

// Node returns the [Node] produced by the [Builder].
//
// If a build error has occurred, it returns the empty [Node], which corresponds to JSON null.
func (b *Builder) Node() Node {
	if !b.Ok() {
		return nullNode
	}

	return b.n
}

func (b *Builder) From(n Node) *Builder {
	b.resetNode()
	b.n = n

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
		b.err = fmt.Errorf("can't clear the children of a non-container node. Node kind is %v", b.n.kind)

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
		b.err = fmt.Errorf("can't swap the children of a non-container node. Node kind is %v", b.n.kind)

		return b
	}

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

	if b.n.kind != nodes.KindObject {
		b.err = fmt.Errorf("can't add a key to a non-object node. Node kind is %v", b.n.kind)

		return b
	}

	value.key = stores.MakeInternedKey(key)
	if _, ok := b.n.keysIndex[value.key]; ok {
		b.err = fmt.Errorf("key is already present in object: %q", key)

		return b
	}

	b.n.children = append(b.n.children, value)
	b.n.keysIndex[value.key] = len(b.n.children)

	return b
}

func (b *Builder) PrependKey(key string, value Node) *Builder {
	if !b.Ok() {
		return b
	}

	if b.n.kind != nodes.KindObject {
		b.err = fmt.Errorf("can't prepend a key into a non-object node. Node kind is %v", b.n.kind)

		return b
	}

	value.key = stores.MakeInternedKey(key)

	if _, ok := b.n.keysIndex[value.key]; ok {
		b.err = fmt.Errorf("key is already present in object: %q", key)

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

	if b.n.kind != nodes.KindObject {
		b.err = fmt.Errorf("can't insert a key into a non-object node. Node kind is %v", b.n.kind)

		return b
	}

	if position <= 0 {
		return b.PrependKey(key, value)
	}

	if position >= len(b.n.children) {
		return b.AppendKey(key, value)
	}

	if _, ok := b.n.keysIndex[value.key]; ok {
		b.err = fmt.Errorf("key is already present in object: %q", key)

		return b
	}

	value.key = stores.MakeInternedKey(key)
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

	if b.n.kind != nodes.KindObject {
		b.err = fmt.Errorf("can't remove a key from a non-object node. Node kind is %v", b.n.kind)

		return b
	}

	k := stores.MakeInternedKey(key)
	index, ok := b.n.keysIndex[k]
	if !ok {
		// key is not present: no error
		return b
	}

	delete(b.n.keysIndex, k)
	b.n.children = slices.Delete(b.n.children, index, index+1)

	return b
}

func (b *Builder) AppendElem(value Node) *Builder {
	if !b.Ok() {
		return b
	}

	if b.n.kind != nodes.KindArray {
		b.err = fmt.Errorf("can't add an element to a non-array node. Node kind is %v", b.n.kind)

		return b
	}

	b.n.children = append(b.n.children, value)

	return b
}

func (b *Builder) PrependElem(value Node) *Builder {
	if !b.Ok() {
		return b
	}

	if b.n.kind != nodes.KindArray {
		b.err = fmt.Errorf("can't add an element to a non-array node. Node kind is %v", b.n.kind)

		return b
	}

	b.n.children = slices.Insert(b.n.children, 0, value)

	return b
}

func (b *Builder) InsertElem(position int, value Node) *Builder {
	if !b.Ok() {
		return b
	}

	if b.n.kind != nodes.KindArray {
		b.err = fmt.Errorf("can't add an element to a non-array node. Node kind is %v", b.n.kind)

		return b
	}

	if position < 0 {
		return b.PrependElem(value)
	}

	if position >= len(b.n.children) {
		return b.AppendElem(value)
	}

	b.n.children = slices.Insert(b.n.children, position, value)

	return b
}

func (b *Builder) RemoveElem(position int) *Builder {
	if !b.Ok() {
		return b
	}

	if b.n.kind != nodes.KindArray {
		b.err = fmt.Errorf("can't remove an element from a non-array node. Node kind is %v", b.n.kind)

		return b
	}
	if position >= len(b.n.children) || position < 0 {
		b.err = fmt.Errorf("can't remove an out of range element. %d >= %d", position, len(b.n.children))

		return b
	}

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

	b.n.value = b.s.PutValue(stores.MakeStringValue(value))

	return b
}

func (b *Builder) BytesValue(value []byte) *Builder {
	if !b.Ok() {
		return b
	}

	b.n.kind = nodes.KindScalar
	b.resetNode()

	b.n.value = b.s.PutValue(stores.MakeScalarValue(token.MakeWithValue(token.String, value)))

	return b
}

func (b *Builder) BoolValue(value bool) *Builder {
	if !b.Ok() {
		return b
	}

	b.n.kind = nodes.KindScalar
	b.resetNode()

	b.n.value = b.s.PutBool(value)

	return b
}

func (b *Builder) NumberValue(value types.Number) *Builder {
	if !b.Ok() {
		return b
	}

	b.n.kind = nodes.KindScalar
	b.resetNode()

	b.n.value = b.s.PutValue(stores.MakeNumberValue(value))

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
	default:
		// TODO set err
		panic("yay")
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

	return b
}

func (b *Builder) resetNode() {
	if b.n.children != nil {
		b.n.children = b.n.children[:0]
	}
	clear(b.n.keysIndex)
}

func buildFromFloat[T conv.Float](b *Builder, value T) *Builder {
	b.n.kind = nodes.KindScalar
	b.resetNode()
	b.n.value = b.s.PutValue(stores.MakeFloatValue(value))

	return b
}

func buildFromInteger[T conv.Signed](b *Builder, value T) *Builder {
	b.n.kind = nodes.KindScalar
	b.resetNode()
	b.n.value = b.s.PutValue(stores.MakeIntegerValue(value))

	return b
}

func buildFromUinteger[T conv.Unsigned](b *Builder, value T) *Builder {
	b.n.kind = nodes.KindScalar
	b.resetNode()
	b.n.value = b.s.PutValue(stores.MakeUintegerValue(value))

	return b
}
