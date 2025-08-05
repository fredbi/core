package borrow

import (
	"io"

	"github.com/fredbi/core/json/internal"
	"github.com/fredbi/core/json/writers"
)

type Encoder interface {
	writerToWriterFactory(io.Writer) (writers.StoreWriter, func())
	encodeStore(writers.StoreWriter) error
}

func BorrowAppendText[T Encoder](d T, b []byte) ([]byte, error) {
	w := internal.BorrowAppendWriter()
	w.Set(b)
	jw, redeem := d.writerToWriterFactory(w)
	defer func() {
		internal.RedeemAppendWriter(w)
		redeem()
	}()

	err := d.encodeStore(jw)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

func BorrowEncodeBytes[T Encoder](d T) ([]byte, error) {
	buf := internal.BorrowBytesBuffer()
	jw, redeem := d.writerToWriterFactory(buf)
	defer func() {
		internal.RedeemBytesBuffer(buf)
		redeem()
	}()

	if err := d.encode(jw); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
