package types

import "github.com/fredbi/core/swag/pools"

// Resettable is implemented by types that can reset their state.
//
// This is useful when working with pools.
type Resettable = pools.Resettable

// WithErrState is the common interface for all types that manage an internal error state.
//
// This is useful to descend a hierarchical structure without stacking return errors.
type WithErrState interface {
	Ok() bool
	Err() error
}

// ErrSetter is the common interface for all types that accept that callers may override their internal error state.
type ErrStateSetter interface {
	SetErr(error)
}

// TODO: should be from reader
type BytesLoaderFunc func(string) ([]byte, error)

// DocumentShareable allows a [stores.Store] object to share services across the [json.Document]s it holds.
// TODO: don't know yet
type DocumentShareable interface {
	// Loader function to grab JSON from a remote or local file location.
	Loader() BytesLoaderFunc

	// RefCache acts as a searchable index of documents loaded from a $ref URL
	// RefCache(string) // TODO
}
