package json

import (
	"bytes"

	"github.com/fredbi/core/swag/pools"
)

var (
	poolOfBuilders      = pools.New[Builder]()
	poolOfDocuments     = pools.New[Document]()
	poolOfBuffers       = pools.New[bytes.Buffer]()
	poolOfAppendWriters = pools.New[appendWriter]()
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
