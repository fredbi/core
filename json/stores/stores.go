package stores

import (
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores/values"
	"github.com/fredbi/core/json/types"
	"github.com/fredbi/core/json/writers"
)

// Handle distributed by a [Store], which corresponds to some [values.Value].
type Handle uint64

// HandleZero is the zero value of a [Handle], which corresponds to the "null" JSON value.
const HandleZero Handle = 0

// Resolve this [Handle] against a [Store].
//
// This is equivalent to calling [Store.Get] with this [Handle].
func (h Handle) Resolve(s Store) values.Value {
	return s.Get(h)
}

// Store is the interface for JSON value stores.
//
// A [Store] knows how to hold the scalar values in a JSON document.
//
// [Store] doesn't have to handle errors as it present a closed, error-free interface.
//
// Corrupted input such as a hand-crafted invalid [Handle] or corrupted [Value] results in a panic.
type Store interface {
	// Get a [Value] given a [Handle].
	//
	// Whenever used in the context of a [VerbatimStore], a [Handle] representing non-significant blank space
	// returns a string [Value].
	//
	// Notice that the default store provides some options to control how the returned value is allocated.
	Get(Handle) values.Value

	// WriteTo writes the value pointed to by the [Handle] to a [writers.StoreWriter].
	//
	// Using [Store.WriteTo] avoids to create an intermediary [values.Value], if the intent is just to write it.
	//
	// Whenever used in the context of a [VerbatimStore], a [Handle] representing non-significant blank space
	// is written down as-is (and not as a JSON string).
	WriteTo(writers.StoreWriter, Handle)

	// Put in the [Store] a new value from a token [token.T] and return the inner [Handle].
	//
	// Token values are copied into the [Store], and it is safe to reuse the provided [token.T].
	//
	// Only tokens representing a scalar value are  allowed: string, number, bool or null.
	//
	// [Store.PutToken] should panic for invalid tokens such as a separator.
	PutToken(token.T) Handle

	// Put a [values.Value] in the [Store] and return the inner [Handle].
	PutValue(values.Value) Handle

	// PutNull puts a null value into the [Store].
	//
	// This is equivalent to [Store.PutValue] with a null [values.Value].
	PutNull() Handle

	// PutBool puts a bool value into the [Store].
	//
	// This is equivalent to [Store.PutValue] with a boolean [values.Value].
	PutBool(bool) Handle

	// Len yields the current size of the [Store] inner memory arena.
	//
	// Notice that the [Store] does not necessarily store all [Handle] s in the arena.
	Len() int

	// Resettable means the [Store] knows how to reset its internal state so as to be reused.
	types.Resettable
}
