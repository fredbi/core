package providers

import (

	// TODO: should alias token kinds somehow to avoid spreading this

	"github.com/fredbi/core/jsonschema/analyzers/structural"
	"github.com/fredbi/core/swag/mangling"
)

// NameProvider provides go names for identifiers, files and packages created from schemas.
//
// The [NameProvider] is not intended for concurrent use: internally, data structures are maintained
// to ensure that no conflicting names are produced.
//
// TODO(fred): consider naming strategy based on title if present
// TODO(fred): refactor to reduce cognitive complexity of this function
type NameProvider struct {
	options

	filesNamespaces map[string]map[string]struct{}
}

// NewNameProvider builds a new [NameProvider] with possible options.
func NewNameProvider(opts ...Option) *NameProvider {
	p := NameProvider{
		options:         optionsWithDefaults(opts),
		filesNamespaces: make(map[string]map[string]struct{}),
	}
	p.mangler = mangling.New(p.manglingOptions...)

	return &p
}

func noaudit(_ structural.AuditAction, _ string) {}
