package registries

import (
	"maps"
	"slices"

	"github.com/fredbi/core/strfmt"
)

// Registry holds a collection of recognized JSON schema string formats to be parsed or just validated.
//
// [Registry] is intended to be used by APIs at runtime, not during analysis or code generation.
type Registry interface {
	// Parse and validate a string for a given format.
	Parse(format string, value string) (strfmt.Format, error)
	// Validate a string for a given format.
	Validate(format string, value string) error

	// SupportedFormats provides the list of all formats recognized by this [Registry]
	SupportedFormats() []string
}

// Merge returns a merged [Registry] built out of a list of registries.
//
// If a format is supported by several registries, the last provided one will be used to
// resolve that format.
func Merge(registries ...Registry) Registry {
	return MakeCompoundRegistry(registries)
}

func MakeCompoundRegistry(merged []Registry) CompoundRegistry {
	r := CompoundRegistry{
		index: make(map[string]Registry),
	}

	for _, registry := range merged {
		for _, format := range registry.SupportedFormats() {
			r.index[format] = registry
		}
	}

	return r
}

// CompoundRegistry is a [Registry] built from [Merge].
type CompoundRegistry struct {
	index map[string]Registry
}

func (r CompoundRegistry) Parse(format string, value string) (strfmt.Format, error) {
	ir, ok := r.index[format]
	if !ok {
		return nil, ErrNotFound(format)
	}

	return ir.Parse(format, value)
}

func (r CompoundRegistry) Validate(format string, value string) error {
	ir, ok := r.index[format]
	if !ok {
		return ErrNotFound(format)
	}

	return ir.Validate(format, value)
}

func (r CompoundRegistry) SupportedFormats() []string {
	return slices.Sorted(maps.Keys(r.index))
}

func (r CompoundRegistry) Index(format string) (Registry, bool) {
	v, ok := r.index[format]

	return v, ok
}
