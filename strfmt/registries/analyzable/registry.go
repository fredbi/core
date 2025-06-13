package analyzable

import (
	"github.com/fredbi/core/json"
	"github.com/fredbi/core/strfmt/registries"
)

// Registry holds a collection of recognized JSON schema string formats that
// may be consumed when analyzing a JSON schema.
//
// [Registry] is like [registries.Registry], and is intended to be used in the context of the analysis of a JSON schema.
//
// For runtime support, see [registries.Registry].
type Registry interface {
	registries.Registry

	// ImpliedValidations returns a minimal set of standard JSON validations that a given format
	// must verify.
	//
	// This set of validations do not replace the custom validator for the format, but may be used
	// by an analyzer to infer minimal constraints such as minimum or maximum length or regexp.
	//
	// This is used by the validation analyzer to evaluate and simplify compound validations.
	//
	// If the format is unsupported, an empty [json.Document] is returned.
	//
	// Example:
	//
	//  ImpliedValidations("date") ->
	// {
	//   "$comment": "A RFC3339 date is a string of exactly 10 bytes with date formatted as yyyy-mm-dd",
	//   "type": "string",
	//   "minLength": 10,
	//   "maxLength": 10,
	//   "pattern": "^\\d{4}-\\d{2}-\\d{2}" // TODO: digit pattern can be made more accurate
	// }
	//
	// In the above example, a smart validation analyzer would be able to simplify redundant of mutually exclusive validations such as:
	//
	// date:
	//   type: string
	//   format: date
	//   minLength: 1  --> redundant
	//
	// date:
	//   type: string
	//   format: date
	//   pattern: "^D-\.*" --> incompatible
	//
	ImpliedValidations(format string) json.Document
}

type compoundRegistry struct {
	registries.CompoundRegistry
}

func (r compoundRegistry) ImpliedValidations(format string) json.Document {
	ir, ok := r.Index(format)
	if !ok {
		return json.EmptyDocument
	}

	ar := ir.(Registry)

	return ar.ImpliedValidations(format)
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
