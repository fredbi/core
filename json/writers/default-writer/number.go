package writer

import (
	"encoding"
	"fmt"
	"math/big"

	"github.com/fredbi/core/swag/conv"
)

// Number writes a number from any native numerical go type, except complex numbers.
//
// Types from the math/big package are supported: [big.Int], [big.Rat], [big.Float].
//
// Numbers provided as a slice of bytes are also supported (no check is carried out).
//
// The method panics if the argument is not a numerical type or []byte.
func (w *W) Number(v any) {
	if w.err != nil {
		return
	}

	switch n := v.(type) {
	case uint8:
		w.buffer.WriteString(conv.FormatUinteger(n))
	case uint16:
		w.buffer.WriteString(conv.FormatUinteger(n))
	case uint32:
		w.buffer.WriteString(conv.FormatUinteger(n))
	case uint64:
		w.buffer.WriteString(conv.FormatUinteger(n))
	case uint:
		w.buffer.WriteString(conv.FormatUinteger(n))
	case int8:
		w.buffer.WriteString(conv.FormatInteger(n))
	case int16:
		w.buffer.WriteString(conv.FormatInteger(n))
	case int32:
		w.buffer.WriteString(conv.FormatInteger(n))
	case int64:
		w.buffer.WriteString(conv.FormatInteger(n))
	case int:
		w.buffer.WriteString(conv.FormatInteger(n))
	case float32:
		w.buffer.WriteString(conv.FormatFloat(n))
	case float64:
		w.buffer.WriteString(conv.FormatFloat(n))
	case []byte:
		// TODO: check  // TODO: case string
		w.buffer.WriteBinary(n)
	case *big.Int:
		if n == nil {
			return
		}
		w.append(n)
		return
	case big.Int:
		w.append(&n)
		return
	case *big.Rat:
		if n == nil {
			return
		}
		f, _ := n.Float64()
		w.buffer.WriteString(conv.FormatFloat(f))
	case big.Rat:
		f, _ := n.Float64()
		w.buffer.WriteString(conv.FormatFloat(f))
	case *big.Float:
		if n == nil {
			return
		}
		w.append(n)
		return
	case big.Float:
		w.append(&n)
		return
	default:
		panic(fmt.Errorf(
			"expected argument to Number() to be of a numerical type, but got: %T: %w",
			v, ErrDefaultWriter,
		))
	}

	w.err = w.buffer.Err()
}

func (w *W) append(n encoding.TextAppender) {
	buf, redeem := poolOfNumberBuffers.BorrowWithRedeem()
	defer redeem()
	b := buf.Slice()

	b, err := n.AppendText(b)
	if err != nil {
		w.err = err

		return
	}
	w.buffer.WriteBinary(b)
	w.err = w.buffer.Err()
}
