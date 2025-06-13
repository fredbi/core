package models

import (
	"fmt"

	"github.com/fredbi/core/json/dynamic"
)

// MapExtensionForType validates and maps extensions that affects schema generation.
//
// It is provided as a callback executed during the schema analysis.
//
// Additional extensions that alter naming and layout (e.g. x-go-name, x-go-package, etc.) are handled by [providers.NameProvider].
func (p *Generator) MapExtensionForType(directive string, jazon dynamic.JSON) (any, error) {
	switch directive {
	case "x-go-type":
		ext := jazon.Interface()
		asString, ok := ext.(string)
		if !ok {
			return nil, fmt.Errorf("invalid %s directive: argument should be a string, but got %T", directive, ext) // TODO: the analyzer should wrap this with some context: add line number etc
		}
		return asString, nil
	case "x-go-nullable", "x-nullable", "x-pointer", "x-go-omitempty":
		ext := jazon.Interface()
		asBool, ok := ext.(bool)
		if !ok {
			return nil, fmt.Errorf("invalid %s directive: argument should be a bool, but got %T", directive, ext) // TODO: add line number etc
		}
		return asBool, nil
	default:
		return jazon, nil
	}
}
