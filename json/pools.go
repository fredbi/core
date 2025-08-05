package json

import (
	"github.com/fredbi/core/swag/pools"
)

// TODO: move to internal, so we can export and document.

var (
	poolOfBuilders  = pools.New[Builder]()
	poolOfDocuments = pools.New[Document]()
)

func BorrowBuilder() *Builder {
	return poolOfBuilders.Borrow()
}

func RedeemBuilder(b *Builder) {
	poolOfBuilders.Redeem(b)
}

func BorrowDocument() *Document {
	return poolOfDocuments.Borrow()
}

func RedeemDocument(d *Document) {
	poolOfDocuments.Redeem(d)
}
