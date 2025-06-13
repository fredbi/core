package structural

import (
	"maps"

	"github.com/fredbi/core/json/dynamic"
	"github.com/fredbi/core/swag/stringutils"
)

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

type ExtensionMapper func(string, dynamic.JSON) (any, error)

func (m ExtensionMapper) MapExtension(key string, jazon dynamic.JSON) (any, error) {
	return m(key, jazon)
}

func passThroughMapper(_ string, jazon dynamic.JSON) (any, error) {
	return jazon, nil
}
