package internal

import (
	"bytes"

	"github.com/fredbi/core/swag/pools"
)

var (
	poolOfAppendWriters = pools.New[AppendWriter]()
	poolOfBuffers       = pools.New[bytes.Buffer]()
)

func BorrowAppendWriter() *AppendWriter {
	return poolOfAppendWriters.Borrow()
}

func RedeemAppendWriter(w *AppendWriter) {
	poolOfAppendWriters.Redeem(w)
}

func BorrowBytesBuffer() *bytes.Buffer {
	return poolOfBuffers.Borrow()
}

func RedeemBytesBuffer(b *bytes.Buffer) {
	poolOfBuffers.Redeem(b)
}
