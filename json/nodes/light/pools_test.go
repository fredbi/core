package light

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	lexer "github.com/fredbi/core/json/lexers/default-lexer"
	codes "github.com/fredbi/core/json/lexers/error-codes"
	store "github.com/fredbi/core/json/stores/default-store"
	writer "github.com/fredbi/core/json/writers/default-writer"
)

// TestParentContextReset verifies that Reset drops every injected reference, so a ParentContext
// redeemed to the pool never pins a store, lexer, writer, error context, options or path while idle.
func TestParentContextReset(t *testing.T) {
	ctx := &ParentContext{}
	ctx.S = store.New()
	ctx.L = lexer.New(bytes.NewBufferString("1"))
	ctx.W = writer.NewUnbuffered(new(bytes.Buffer))
	ctx.X = "scratch"
	ctx.C = &codes.ErrContext{}
	ctx.DO.tolerateDuplKey = true
	ctx.P = Path{{kind: pathElemInt, i: 3}}

	ctx.Reset()

	assert.Nil(t, ctx.S)
	assert.Nil(t, ctx.L)
	assert.Nil(t, ctx.W)
	assert.Nil(t, ctx.X)
	assert.Nil(t, ctx.C)
	assert.False(t, ctx.DO.tolerateDuplKey)
	assert.Empty(t, ctx.P)
}

// TestBorrowParentContext checks the borrow/redeem round-trip yields a clean context and that the
// PoolRedeemable closure detects a double-redeem.
func TestBorrowParentContext(t *testing.T) {
	ctx, redeem := BorrowParentContext()
	require.NotNil(t, ctx)
	assert.Nil(t, ctx.S)
	assert.Empty(t, ctx.P)

	ctx.X = "dirty"
	redeem()

	t.Run("a second redeem panics", func(t *testing.T) {
		assert.Panics(t, redeem)
	})

	t.Run("a re-borrow is clean", func(t *testing.T) {
		ctx2, redeem2 := BorrowParentContext()
		defer redeem2()
		assert.Nil(t, ctx2.X)
		assert.Nil(t, ctx2.S)
		assert.Empty(t, ctx2.P)
	})
}

// TestBorrowPath checks the path borrow/redeem round-trip and double-redeem detection.
func TestBorrowPath(t *testing.T) {
	p, redeem := BorrowPath()
	assert.Empty(t, p)

	redeem()
	assert.Panics(t, redeem)
}
