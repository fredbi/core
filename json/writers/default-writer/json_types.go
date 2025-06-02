package writer

import "github.com/fredbi/core/json/types"

func (w *W) JSONString(value types.String) {
	w.StringBytes(value.Value)
}

func (w *W) JSONNumber(value types.Number) {
	w.Raw(value.Value)
}

func (w *W) JSONBoolean(value types.Boolean) {
	w.Bool(value.Value)
}
