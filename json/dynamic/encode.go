package dynamic

import (
	"fmt"
	"math/big"

	"github.com/fredbi/core/json/writers"
)

func (d *JSON) encode(w writers.Writer) {
	if !w.Ok() {
		return
	}

	if d.inner == nil {
		w.Null()

		return
	}

	switch inner := d.inner.(type) {
	case map[string]any:
		w.StartObject()
		if !w.Ok() {
			return
		}

		l := len(inner)

		if l == 0 {
			w.EndObject()

			return
		}

		i := 0
		for key, value := range inner {
			w.String(key)

			v := JSON{inner: value}
			v.encode(w)
			if i < l {
				w.Comma()
			}
			i++
		}

		w.EndObject()

		return

	case []any:
		w.StartArray()
		if !w.Ok() {
			return
		}

		l := len(inner)
		if l == 0 {
			w.EndArray()

			return
		}

		v0 := JSON{inner: inner[0]}
		v0.encode(w)

		for _, elem := range inner[1:] {
			w.Comma()
			v := JSON{inner: elem}
			v.encode(w)
		}

		w.EndArray()

		return

	case string:
		w.String(inner)
	case bool:
		w.Bool(inner)
	case float64:
		w.Number(inner)
	case float32:
		w.Number(inner)
	case int8:
		w.Number(inner)
	case int16:
		w.Number(inner)
	case int32:
		w.Number(inner)
	case int64:
		w.Number(inner)
	case int:
		w.Number(inner)
	case uint8:
		w.Number(inner)
	case uint16:
		w.Number(inner)
	case uint32:
		w.Number(inner)
	case uint64:
		w.Number(inner)
	case uint:
		w.Number(inner)
	case *big.Int:
		w.Number(inner)
	case big.Int:
		w.Number(inner)
	case *big.Rat:
		w.Number(inner)
	case big.Rat:
		w.Number(inner)
	case *big.Float:
		w.Number(inner)
	case big.Float:
		w.Number(inner)
	// TODO: json types, strformat types, ordered map
	default:
		w.SetErr(fmt.Errorf("invalid dynamic JSON type: %T", d.inner))
		return
	}
}
