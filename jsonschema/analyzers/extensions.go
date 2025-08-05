package analyzers

import (
	"maps"

	"github.com/fredbi/core/swag/stringutils"
)

// Extensions is a map of "x-*" extensions, as commonly defined in OpenAPI objects.
//
// TODO: declare this type in some common place but not json.
type Extensions map[string]any // x-...

func (e Extensions) Add(key string, value any) {
	e[key] = value
}

func (e Extensions) Has(extension string, aliases ...string) bool {
	return stringutils.MapContains(e, append([]string{extension}, aliases...)...)
}

func (e Extensions) Merge(merged Extensions) {
	maps.Copy(e, merged)
}

func (e Extensions) Get(extension string, aliases ...string) (any, bool) {
	value, ok := e[extension]
	if ok {
		return value, ok
	}

	for _, alias := range aliases {
		value, ok = e[alias]
		if ok {
			return value, ok
		}
	}

	return nil, false
}
