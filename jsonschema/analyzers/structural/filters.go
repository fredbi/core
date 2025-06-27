package structural

import "github.com/fredbi/core/jsonschema/analyzers/structural/order"

// Filter alters the output of [SchemaAnalyzer.Schemas].
type Filter func(f *filters)

type filters struct {
	WantsNamed      bool
	WantsEnum       bool
	ExcludeEnum     bool
	WantsRef        bool // more or less equivalent to
	Ordering        order.SchemaOrdering
	WantsOnlyLeaves bool
	FilterFunc      func(AnalyzedSchema) bool
	PkgFilterFunc   func(AnalyzedPackage) bool
}

func applyFiltersWithDefault(opts []Filter) filters {
	f := filters{
		Ordering: order.TopDown,
	}

	for _, apply := range opts {
		apply(&f)
	}

	return f
}

// OnlyNamedSchemas keeps only schemas with a name.
//
// Anonymous schemas are filtered out.
func OnlyNamedSchemas() Filter {
	return func(f *filters) {
		f.WantsNamed = true
	}
}

// OnlyEnumSchemas keeps only schemas with an "enum" validation.
func OnlyEnumSchemas() Filter {
	return func(f *filters) {
		f.WantsEnum = true
		f.ExcludeEnum = false
	}
}

// ExcludeEnumSchemas filters out all schemas with an "enum" validation.
func ExcludeEnumSchemas(excluded bool) Filter {
	return func(f *filters) {
		f.ExcludeEnum = excluded
		if excluded {
			f.WantsEnum = false
		}
	}
}

// OnlyRefSchemas keeps only schemas that contain a "$ref" key.
func OnlyRefSchemas() Filter {
	return func(f *filters) {
		f.WantsRef = true
	}
}

// WithOrderedSchemas imposes an output ordering.
func WithOrderedSchemas(o order.SchemaOrdering) Filter {
	return func(f *filters) {
		f.Ordering = o
	}
}

// OnlyLeaves keeps only schemas without dependencies,
// i.e. leaves in the dependency tree.
func OnlyLeaves() Filter {
	return func(f *filters) {
		f.WantsOnlyLeaves = true
	}
}

// WithFilterFunc applies the provided function on schemas, and keeps
// only those that yield true.
func WithFilterFunc(fn func(AnalyzedSchema) bool) Filter {
	return func(f *filters) {
		f.FilterFunc = fn
	}
}

func WithPackageFilterFunc(fn func(AnalyzedPackage) bool) Filter {
	return func(f *filters) {
		f.PkgFilterFunc = fn
	}
}
