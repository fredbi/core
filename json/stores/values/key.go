package values

import "unique"

// InternedKey is a key in a JSON object, interned in memory.
//
// The current implementation of [InternedKey] merely wraps the standard [unique] package.
// Therefore, interning occurs on global memory.
type InternedKey struct {
	unique.Handle[string]
}

// String representation of the [InternedKey].
func (k InternedKey) String() string {
	return k.Value()
}

// MakeInternedKey builds a handle for an [InternedKey] string.
func MakeInternedKey(s string) InternedKey {
	return InternedKey{
		Handle: unique.Make(s),
	}
}
