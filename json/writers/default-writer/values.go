package writer

import (
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores/values"
)

func nullToken() []byte { return []byte("null") }

// Value writes a [values.Value]
func (w *W) Value(v values.Value) {
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

	w.err = w.buffer.Err()
}

// Null writes a null token ("null").
func (w *W) Null() {
	if w.err != nil {
		return
	}

	w.buffer.WriteBinary(nullToken())
	w.err = w.buffer.Err()
}

// Key write a key [values.InternedKey] followed by a colon (":").
func (w *W) Key(key values.InternedKey) {
	w.String(key.String())
	w.Colon()
}
