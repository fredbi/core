package providers

import (
	"github.com/fredbi/core/mangling"
)

const (
	indexingStart = 2   // when adopting an indexing strategy for an identifier, the minimum index
	maxAttempts   = 100 // when adopting an indexing strategy for an identifier, the max sensible number of iterations supported
)

// NameProvider provides go names for identifiers, files and packages created from schemas.
//
// The [NameProvider] is not intended for concurrent use: internally, data structures are maintained
// to ensure that no conflicting names are produced.
type NameProvider struct {
	options

	files map[string]map[string]struct{}
}

// NewNameProvider builds a new [NameProvider] with possible options.
func NewNameProvider(opts ...Option) *NameProvider {
	p := NameProvider{
		options: optionsWithDefaults(opts),
		files:   make(map[string]map[string]struct{}),
	}
	p.mangler = mangling.New(p.manglingOptions...)

	return &p
}
