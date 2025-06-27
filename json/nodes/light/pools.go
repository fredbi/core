//nolint:gochecknoglobals
package light

import (
	"github.com/fredbi/core/json/stores"
	"github.com/fredbi/core/swag/pools"
)

var (
	poolOfBuilders       = pools.New[Builder]()
	poolOfParentContexts = pools.New[ParentContext]()
)

// BorrowBuilder borrows a [Builder] from the pool.
func BorrowBuilder(s stores.Store) *Builder {
	b := poolOfBuilders.Borrow()
	b.s = s

	return b
}

// RedeemBuilder redeems a [Builder] to the pool.
//
// The relinquished [Builder] may be recycled by subsequent calls to [BorrowBuilder].
func RedeemBuilder(b *Builder) {
	poolOfBuilders.Redeem(b)
}

func BorrowParentContext() *ParentContext {
	return poolOfParentContexts.Borrow()
}

func RedeemParentContext(p *ParentContext) {
	poolOfParentContexts.Redeem(p)
}
