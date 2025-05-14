package writer

import (
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores"
)

func (w *W) Value(v stores.Value) {
	switch v.Kind() {
	case token.String:
		w.StringBytes(v.StringValue().Value)
	case token.Number:
		w.NumberBytes(v.NumberValue().Value)
	case token.Boolean:
		w.Bool(v.Bool())
	case token.Null:
		w.Null()
	default:
		// skip
	}
}

// TODO: func (w *W) VerbatimValue(v stores.VerbatimValue) {}

// Null writes a null value.
func (w *W) Null() {
	w.buffer.EnsureSpace(4)
	w.buffer.Buf = append(w.buffer.Buf, null...)
}

func (w *W) Key(key stores.InternedKey) {
	w.String(key.String())
}

func (w *W) Number(v any) {
	/* TODO
	switch n := v.(type) {
	case uint8:
		fallthrough
	case uint16:
		fallthrough
	case uint32:
		fallthrough
	case uint64:
		fallthrough
	case uint:
		fallthrough
	case int8:
		fallthrough
	case int16:
		fallthrough
	case int32:
		fallthrough
	case int64:
		fallthrough
	case int:
		fallthrough
	case float32:
		fallthrough
	case float64:
		fallthrough
	case []byte:
		fallthrough
	default:
		panic("yay")
	}
	*/
}
