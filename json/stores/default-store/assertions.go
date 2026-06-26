package store

import (
	"fmt"

	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/writers"
)

// code assertions are enforced to check input (handles or values).
//
// unlike guards, they operate on user input and are not disabled with a build tag.

// assertOffsetInArena verifies that the offset is not out of range of the arena.
//
// This is a guard against a corrupted incoming [stores.Handle].
func assertOffsetInArena(offset, length int) {
	if offset >= length {
		panic(fmt.Errorf("out of range offset: %d >= %d: %w", offset, length, ErrStore))
	}
}

// assertValidHeader verifies that the header in the [stores.Handle] is valid.
//
// This is a guard against a corrupted incoming [stores.Handle].
func assertValidHeader(header uint8) {
	panic(fmt.Errorf("invalid header in handle: %x: %w", header, ErrStore))
}

// writeToOffsetInArena is the non-panicking counterpart of [assertOffsetInArena] for the writer-driven
// [Store.WriteTo] path. On a corrupted handle it routes the error through the writer's error channel
// (and returns false) instead of panicking, so a corrupted handle surfaces as an error to the caller.
//
// The value-returning paths ([Store.Get], [Store.AppendValueBytes]) have no error sink and keep the
// panic-on-corruption contract; converting those is a separate, store-wide decision.
func writeToOffsetInArena(writer writers.StoreWriter, offset, length int) bool {
	if offset >= length {
		writer.SetErr(fmt.Errorf("out of range offset: %d >= %d: %w", offset, length, ErrStore))

		return false
	}

	return true
}

// writeToInvalidHeader is the non-panicking counterpart of [assertValidHeader] for [Store.WriteTo].
func writeToInvalidHeader(writer writers.StoreWriter, header uint8) {
	writer.SetErr(fmt.Errorf("invalid header in handle: %x: %w", header, ErrStore))
}

// assertValidToken verifies that the passed token holds a valid value, i.e. not a delimiter token or EOF.
func assertValidToken(tok token.T) {
	panic(
		fmt.Errorf(
			"invalid token kind passed to PutToken. Must be a scalar value. Got %v: %w",
			tok.Kind(),
			ErrStore,
		),
	)
}
