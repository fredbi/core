package stores

import (
	"github.com/fredbi/core/json/lexers/token"
	"github.com/fredbi/core/json/stores/values"
)

// VerbatimStore is like [Store], and supports verbatim tokens [token.VT], [VerbatimHandle] s,
// and [VerbatimValue] s.
//
// It adds the capability to store aside non-significant blank space strings in a memory-efficient way.
//
// The [VerbatimStore] may be used to store and reconstruct JSON documents unaltered.
type VerbatimStore interface {
	// GetVerbatim returns a [VerbatimValue] given a [VerbatimHandle].
	//
	// These are actually pairs of [Handle] s and [Value] s.
	GetVerbatim(VerbatimHandle) values.VerbatimValue

	// PutVerbatimToken returns a pair of [Handle] s, in the order of appearance in the JSON stream.
	//
	// It yields the [Handle] to any non-significant blank space appearing before the token value, and the [Handle]
	// to the value itself.
	//
	// A [VerbatimStore] does not store the association between these handles after the verbatim token is split in two parts.
	// Rather, the association is kept by the [VerbatimHandle], which is actually a pair of [Handle] s.
	//
	// This is similar in concept to the [Store] not keeping track of the structure of a JSON document, only values.
	PutVerbatimToken(token.VT) VerbatimHandle

	// Put a [VerbatimValue] and return the inner [VerbatimHandle].
	PutVerbatimValue(values.VerbatimValue) VerbatimHandle

	// PutBlanks returns a [Handle] to a slice of blank characters.
	//
	// It panics if non-blank characters are passed.
	PutBlanks([]byte) Handle

	// [VerbatimStore] extends [Store].
	Store
}

// VerbatimHandle represents a pair of [Handle] s.
//
// The first [Handle] represent the non-significant blank part occurring before a value,
// and the second one the value itself.
type VerbatimHandle struct {
	blanks Handle
	value  Handle
}

// Blanks returns a [Handle] to the non-significant blanks before the value token.
func (v VerbatimHandle) Blanks() Handle {
	return v.blanks
}

// Value returns a [Handle] representing a value in the [Store].
func (v VerbatimHandle) Value() Handle {
	return v.value
}

// MakeVerbatimHandle builds a [VerbatimHandle] from a pair of handles.
//
// These handles represent the non-significant blanks part and the value part respectively.
func MakeVerbatimHandle(blanks, value Handle) VerbatimHandle {
	return VerbatimHandle{
		blanks: blanks,
		value:  value,
	}
}
