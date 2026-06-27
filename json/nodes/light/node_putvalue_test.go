package light

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	lexer "github.com/fredbi/core/json/lexers/default-lexer"
	"github.com/fredbi/core/json/lexers/token"
	nodecodes "github.com/fredbi/core/json/nodes/error-codes"
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/json/types"
	"github.com/fredbi/core/json/writers"
	writer "github.com/fredbi/core/json/writers/default-writer"
)

// zeroStore is a [stores.Store] that rejects every value by returning the zero [stores.Handle].
//
// It exercises the caller-side guards (E-series): the store presents an error-free interface, so a
// rejection surfaces only as a zero handle, which the builder and the decoder must catch and stop on.
type zeroStore struct{}

func (zeroStore) Get(stores.Handle) values.Value { return values.UndefinedValue }

func (zeroStore) AppendValueBytes(dst []byte, _ stores.Handle) (values.Value, []byte) {
	return values.UndefinedValue, dst
}

func (zeroStore) WriteTo(writers.StoreWriter, stores.Handle) {}
func (zeroStore) PutToken(token.T) stores.Handle             { return stores.HandleZero }
func (zeroStore) PutValue(values.Value) stores.Handle        { return stores.HandleZero }
func (zeroStore) PutNull() stores.Handle                     { return stores.HandleZero }
func (zeroStore) PutBool(bool) stores.Handle                 { return stores.HandleZero }
func (zeroStore) Len() int                                   { return 0 }
func (zeroStore) Reset()                                     {}

// TestBuilderZeroHandleGuard checks that when the store rejects a value (zero handle), the builder
// records an error and short-circuits instead of silently producing a node bound to HandleZero.
func TestBuilderZeroHandleGuard(t *testing.T) {
	for name, build := range map[string]func(*Builder) *Builder{
		"StringValue": func(b *Builder) *Builder { return b.StringValue("x") },
		"BytesValue":  func(b *Builder) *Builder { return b.BytesValue([]byte("x")) },
		"BoolValue":   func(b *Builder) *Builder { return b.BoolValue(true) },
		"NumberValue": func(b *Builder) *Builder { return b.NumberValue(types.Number{Value: []byte("1")}) },
		"Float":       func(b *Builder) *Builder { return b.NumericalValue(float64(1.5)) },
		"Integer":     func(b *Builder) *Builder { return b.NumericalValue(int64(-3)) },
		"Uinteger":    func(b *Builder) *Builder { return b.NumericalValue(uint64(7)) },
	} {
		t.Run(name+" stops on a zero handle", func(t *testing.T) {
			b := build(NewBuilder(zeroStore{}))

			require.False(t, b.Ok())
			require.Error(t, b.Err())
			assert.ErrorIs(t, b.Err(), nodecodes.ErrBuilder)
			assert.True(t, b.Node().value.IsZero())
		})
	}
}

// TestDecodeZeroHandleGuard checks that when the store rejects a value during decoding, the fault is
// routed through the lexer's error channel rather than yielding a node silently bound to HandleZero.
func TestDecodeZeroHandleGuard(t *testing.T) {
	for name, jazon := range map[string]string{
		"scalar number": `42`,
		"bool":          `true`,
		"null":          `null`,
		"object":        `{"a":1}`,
		"array":         `[1,2]`,
	} {
		t.Run(name+" surfaces a store rejection as a lexer error", func(t *testing.T) {
			r := bytes.NewBufferString(jazon)
			w := new(bytes.Buffer)
			ctx := &ParentContext{
				L: lexer.New(r),
				W: writer.NewUnbuffered(w),
				S: zeroStore{},
			}

			n := Node{}
			require.NotPanics(t, func() { n.Decode(ctx) })

			require.Error(t, ctx.L.Err())
			assert.ErrorIs(t, ctx.L.Err(), nodecodes.ErrNode)
		})
	}
}
