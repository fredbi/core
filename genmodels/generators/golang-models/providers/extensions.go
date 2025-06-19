package providers

import (
	"fmt"

	"github.com/fredbi/core/json/dynamic"
)

// MapExtension maps extensions into known go types.
//
// The supported extensions act as directives to hint the [NameProvider].
//
// The mapping validations are enforced by the analyzer at analyis time, so later processing can rely on a safe typing for known extensions.
//
// # Directives that affect naming and layout
//
// - x-go-name
// - x-go-package
// - x-go-file-name
// - x-go-enums
// - x-go-wants-validation (x-go-validation)
// - x-go-wants-split-validation (x-go-split-validation)
// - x-go-wants-test (x-go-test)
//
// Extra directives generated for audit purpose:
// - x-go-original-name
// - x-go-original-path
// - x-go-namespace-only
//
// NOTE: extensions such as x-go-type, x-go-nullable, x-nullable which alter the behavior of type generation but not
// naming are mapped by a dedicated mapper.
func (p NameProvider) MapExtension(directive string, jazon dynamic.JSON) (any, error) {
	switch directive {

	// string
	case "x-go-name",
		"x-go-package",       // force output folder and package name
		"x-go-file-name",     // force output file name (no extension)
		"x-go-original-name", // write-only; documentary only: JSON name before transform
		"x-go-original-path", // write-only; documentary only: JSON path before transform
		"x-go-tag":           // hint for tag-based bundling strategy
		ext := jazon.Interface()
		asString, ok := ext.(string)
		if !ok {
			// TODO: in the analyzer - add line number etc: context provided by the analyzer when getting an error from thi callback
			return nil, fmt.Errorf(
				"invalid %s directive: argument should be a string, but got %T: %w",
				directive, ext, ErrNameProvider,
			)
		}
		return asString, nil

	// bool
	case "x-go-wants-validation",
		"x-go-validation",
		"x-go-wants-split-validation",
		"x-go-split-validation",
		"x-go-wants-test",
		"x-go-test",
		"x-go-namespace-only":
		ext := jazon.Interface()
		asBool, ok := ext.(bool)
		if !ok {
			return nil, fmt.Errorf(
				"invalid %s directive: argument should be a bool, but got %T: %w",
				directive, ext, ErrNameProvider,
			)
		}
		return asBool, nil
	// others: []any
	case "x-go-enums":
		ext := jazon.Interface()
		asSlice, ok := ext.([]any)
		if !ok {
			return nil, fmt.Errorf(
				"invalid %s directive: argument should be a slice, but got %T: %w",
				directive, ext, ErrNameProvider,
			)
		}
		output := make([]string, 0, len(asSlice))
		for _, elem := range asSlice {
			asString, isString := elem.(string)
			if !isString {
				return nil, fmt.Errorf(
					"invalid %s directive: element in slice should be a string, but got %T: %w",
					directive, elem, ErrNameProvider,
				)
			}
			output = append(output, asString)
		}
		return output, nil
	default:
		return jazon, nil // keep directive, but don't perform any check or conversion
	}
}
