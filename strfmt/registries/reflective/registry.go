package reflective

import (
	"fmt"
	"reflect"

	"github.com/fredbi/core/strfmt/registries"
	"github.com/go-viper/mapstructure/v2"
)

// Registry holds a collection of recognized JSON schema string formats that
// may be consumed when generating go code.
//
// [Registry] is like [registries.Registry], and is intended to be used in the context of code generation from
// JSON schemas.
//
// For runtime support, see [registries.Registry].
type Registry interface {
	registries.Registry

	// GoType returns the type name for this format
	//
	// Example:
	//   GoType("date") -> "date"
	GoType(format string) (string, bool)

	// GoFullType returns the fully qualified type name for this format
	//
	// Example:
	//   GoFullType("date") -> "github.com/fredbi/core/strfmt/formats.Date"
	GoFullType(format string) (string, bool)

	// GoPackage returns the fully qualified package name for this format, e.g. to be used in imports
	//
	// Example:
	//   GoPackage("date") -> "github.com/fredbi/core/strfmt/formats"
	GoPackage(format string) (string, bool)

	// GoValueConstructor returns the call that builds a zero value.
	//
	// Example:
	//   GoValueConstructor("date") -> "MakeDate()"
	GoValueConstructor(format string) (string, bool)

	// GoPointerConstructor returns the call that builds a pointer to the zero value.
	//
	// Example:
	//   GoPointerConstructor("date") -> "NewDate()"
	GoPointerConstructor(format string) (string, bool)

	// ReflectType returns the format type as a [reflect.Type], for use in dynamic constructs that use type reflection.
	ReflectType(format string) (reflect.Type, bool)

	// MapStructureHookFunc provides a decoder for unmarshaling data using mapstructure.
	MapStructureHookFunc() mapstructure.DecodeHookFunc
}

type compoundRegistry struct {
	registries.CompoundRegistry
}

// GoType returns the type name for this format
func (r compoundRegistry) GoType(format string) (string, bool) {
	ir, ok := r.Index(format)
	if !ok {
		return "", false
	}

	rr := ir.(Registry)

	return rr.GoType(format)
}

// GoFullType returns the fully qualified type name for this format
func (r compoundRegistry) GoFullType(format string) (string, bool) {
	ir, ok := r.Index(format)
	if !ok {
		return "", false
	}

	rr := ir.(Registry)

	return rr.GoFullType(format)
}

// GoPackage returns the fully qualified package name for this format
func (r compoundRegistry) GoPackage(format string) (string, bool) {
	ir, ok := r.Index(format)
	if !ok {
		return "", false
	}

	rr := ir.(Registry)

	return rr.GoPackage(format)
}

// GoValueConstructor...
func (r compoundRegistry) GoValueConstructor(format string) (string, bool) {
	ir, ok := r.Index(format)
	if !ok {
		return "", false
	}

	rr := ir.(Registry)

	return rr.GoValueConstructor(format)
}

// GoPointerConstructor...
func (r compoundRegistry) GoPointerConstructor(format string) (string, bool) {
	ir, ok := r.Index(format)
	if !ok {
		return "", false
	}

	rr := ir.(Registry)

	return rr.GoPointerConstructor(format)
}

// ReflectType returns the format type as a [reflect.Type], for use in dynamic constructs that use type reflection.
func (r compoundRegistry) ReflectType(format string) (reflect.Type, bool) {
	ir, ok := r.Index(format)
	if !ok {
		return nil, false
	}

	rr := ir.(Registry)

	return rr.ReflectType(format)
}

// MapStructureHookFunc provides a decoder for unmarshaling data using mapstructure.
//
// TODO(?): resp. marshaling? should not be useful since they are all fmt.Stringer s
func (r compoundRegistry) MapStructureHookFunc() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, obj any) (any, error) {
		if from.Kind() != reflect.String {
			return obj, nil
		}

		data, ok := obj.(string)
		if !ok {
			return nil, fmt.Errorf("failed to cast %+v to string: %w", obj, registries.ErrFormat)
		}

		for _, format := range r.SupportedFormats() {
			tpe, ok := r.ReflectType(format)
			if !ok {
				continue
			}

			if to == tpe {
				return r.Parse(format, data)
			}
		}

		return obj, nil // pass on without action
	}
}

func Merge(merged ...Registry) Registry {
	asRegistries := make([]registries.Registry, len(merged))
	for i := range merged {
		asRegistries[i] = merged[i]
	}

	return compoundRegistry{
		CompoundRegistry: registries.MakeCompoundRegistry(asRegistries),
	}
}
