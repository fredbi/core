package schema

import (
	"fmt"

	"github.com/fredbi/core/json/dynamic"
)

// MapExtension validates and maps extensions that affects schema generation.
//
// It is provided as a callback executed during the schema analysis.
//
// Additional extensions that alter naming and layout (e.g. x-go-name, x-go-package, etc.) are handled by [providers.NameProvider].
//
// # Supported extensions
//
//   - x-go-type: {string}  TODO: could be an object
//   - x-go-nullable (x-nullable): {bool}
//   - x-go-pointer: {bool}
//   - x-go-omitempty: {bool}
func (b *Builder) MapExtension(directive string, jazon dynamic.JSON) (any, error) {
	switch directive {
	case "x-go-type":
		ext := jazon.Interface()
		asString, ok := ext.(string)
		if !ok {
			return nil, fmt.Errorf(
				"invalid %s directive: argument should be a string, but got %T: %w",
				directive, ext, ErrSchema,
			) // TODO: the analyzer should wrap this with some context: add line number etc
		}
		return asString, nil
	case "x-go-nullable", "x-nullable", "x-go-pointer", "x-go-omitempty":
		ext := jazon.Interface()
		asBool, ok := ext.(bool)
		if !ok {
			return nil, fmt.Errorf(
				"invalid %s directive: argument should be a bool, but got %T: %w",
				directive, ext, ErrSchema,
			)
		}
		return asBool, nil
	default:
		return jazon, nil
	}
}
