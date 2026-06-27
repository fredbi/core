package light

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredbi/core/json/lexers"
	lexer "github.com/fredbi/core/json/lexers/default-lexer"
	nodecodes "github.com/fredbi/core/json/nodes/error-codes"
	store "github.com/fredbi/core/json/stores/default-store"
	"github.com/fredbi/core/json/stores/values"
	writer "github.com/fredbi/core/json/writers/default-writer"
)

var errHookStop = errors.New("hook stop")

func newDecodeCtx(jazon string, do DecodeOptions) (*ParentContext, *Node) {
	ctx := &ParentContext{
		L:  lexer.New(bytes.NewBufferString(jazon)),
		W:  writer.NewUnbuffered(new(bytes.Buffer)),
		S:  store.New(),
		DO: do,
	}

	return ctx, &Node{}
}

// TestDecodeErrorPath checks that an error caught while processing a node carries the JSON Pointer
// (RFC 6901) path of the failing node in its [codes.ErrContext].
func TestDecodeErrorPath(t *testing.T) {
	t.Run("nested object key error reports the full pointer", func(t *testing.T) {
		var do DecodeOptions
		do.AfterKey = func(_ *ParentContext, _ lexers.Lexer, key values.InternedKey, _ Node) (bool, error) {
			if key.String() == "c" {
				return false, errHookStop
			}

			return false, nil
		}

		ctx, n := newDecodeCtx(`{"a":{"b":1,"c":2}}`, do)
		n.Decode(ctx)

		require.Error(t, ctx.L.Err())
		require.NotNil(t, ctx.C)
		assert.Equal(t, "/a/c", ctx.C.Path)
		assert.ErrorIs(t, ctx.C.Err, nodecodes.ErrNode)
		assert.ErrorIs(t, ctx.C.Err, errHookStop)
	})

	t.Run("array element error reports the index pointer", func(t *testing.T) {
		var count int
		var do DecodeOptions
		do.AfterElem = func(_ *ParentContext, _ lexers.Lexer, _ Node) (bool, error) {
			count++
			if count == 3 {
				return false, errHookStop
			}

			return false, nil
		}

		ctx, n := newDecodeCtx(`[10,20,30]`, do)
		n.Decode(ctx)

		require.Error(t, ctx.L.Err())
		require.NotNil(t, ctx.C)
		assert.Equal(t, "/2", ctx.C.Path)
	})

	t.Run("key needing JSON Pointer escaping is escaped", func(t *testing.T) {
		var do DecodeOptions
		do.AfterKey = func(_ *ParentContext, _ lexers.Lexer, _ values.InternedKey, _ Node) (bool, error) {
			return false, errHookStop
		}

		ctx, n := newDecodeCtx(`{"a/b~c":1}`, do)
		n.Decode(ctx)

		require.NotNil(t, ctx.C)
		assert.Equal(t, "/a~1b~0c", ctx.C.Path)
	})

	t.Run("happy path leaves no error context", func(t *testing.T) {
		ctx, n := newDecodeCtx(`{"a":[1,2,3]}`, DecodeOptions{})
		n.Decode(ctx)

		require.NoError(t, ctx.L.Err())
		assert.Nil(t, ctx.C)
	})
}
