package writer

import "github.com/fredbi/core/json/types"

// JSONString writes a JSON value of [types.String].
//
// Nothing is written if the value is undefined.
func (w *W) JSONString(value types.String) {
	if w.err != nil || !value.IsDefined() || len(value.Value) == 0 {
		return
	}

	w.buffer.WriteSingleByte('"')
	w.buffer.WriteText(value.Value)
	w.buffer.WriteSingleByte('"')
}

// JSONNumber writes a JSON value of [types.Number].
//
// Nothing is written if the value is undefined.
func (w *W) JSONNumber(value types.Number) {
	if w.err != nil || !value.IsDefined() || len(value.Value) == 0 {
		return
	}

	w.buffer.WriteBinary(value.Value)
}

// JSONBoolean writes a JSON value of [types.Boolean].
//
// Nothing is written if the value is undefined.
func (w *W) JSONBoolean(value types.Boolean) {
	if w.err != nil || !value.IsDefined() {
		return
	}

	w.Bool(value.Value)
}

// JSONNull writes a JSON value of [types.NullType], i.e. the "null" token.
//
// Nothing is written if the value is undefined.
func (w *W) JSONNull(value types.NullType) {
	if w.err != nil || !value.IsDefined() {
		return
	}

	w.Null()
}
