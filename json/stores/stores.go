package stores

import (
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/types"
)

// Handle distributed by a [Store], which corresponds to some [Value]
type Handle uint64

const HandleZero Handle = 0

// Resolve this [Handle] against a [Store].
//
// This is a shorthand to calling [Store].Get with this [Handle].
func (h Handle) Resolve(s Store, opts ...Option) Value {
	return s.Get(h, opts...)
}

// Store is the interface for JSON value stores.
//
// A [Store] knows how to hold the scalar values in a JSON documents.
//
// [Store] doesn't have to handle errors as it present a closed, error-free interface.
//
// Corrupted input such as a hand-crafted invalid [Handle] or corrupted [Value] results in a panic.
type Store interface {
	// Get a [Value] given a [Handle]
	//
	// Whenever used in the context of a [VerbatimStore], a [Handle] representing non-significant blank space
	// returns a string [Value].
	//
	// Using [Option], an optional []byte buffer may be provided by the caller to keep control of any possible inner allocations
	// when creating the returned [Value]. This is useful when the returned [Value] is not intended to be kept and
	// allows the caller to recycle the provided buffer.
	Get(Handle, ...Option) Value

	// Write the value pointed to by the [Handle] directly passed to a [writers.Writer].
	//
	// Whenever used in the context of a [VerbatimStore], a [Handle] representing non-significant blank space
	// is written down as-is (and not as a JSON string).
	Write(Handle)

	// Put the value inside a token [token.T] and return the inner [Handle].
	//
	// Token values are copied into the [Store], and it is safe to reuse the provided [token.T].
	//
	// Only tokens representing a scalar value are  allowed: string, number, bool or null.
	//
	// [Store.PutToken] should panic for invalid tokens such as a separator.
	PutToken(token.T) Handle

	// Put a [Value] and return the inner [Handle].
	PutValue(Value) Handle

	// PutNull puts a null value into the [Store].
	//
	// This is equivalent to [Store.PutValue] with a null [Value].
	PutNull() Handle

	// PutBool puts a bool value into the [Store].
	//
	// This is equivalent to [Store.PutValue] with a boolean [Value].
	PutBool(bool) Handle

	// Len yields the current size of the [Store] inner memory arena.
	Len() int

	// Resettable means the [Store] knows how to reset its internal state so as to be reused.
	types.Resettable
}
