package structural

import (
	"github.com/fredbi/core/jsonschema/analyzers/structural/bundle"
)

// bundleOption configures the behavior of [SchemaAnalyzer.Bundle].
type bundleOptions struct {
	pruneUnused   bool
	withBacktrack bool

	// layout
	bundleStrategy       bundle.SchemaBundlingStragegy
	bundleAggressiveness bundle.SchemaBundlingAggressiveness
	bundleSingleRoot     bool

	// enum layout
	bundleEnums        bool
	bundleEnumsPackage string

	// schema naming
	bundleNameProvider     NameProvider
	bundleNameIdentifier   UniqueIdentifier
	bundleNameDeconflicter Deconflicter

	// package naming
	bundlePathProvider     PackageNameProvider
	bundlePathIdentifier   UniqueIdentifier
	bundlePathDeconflicter Deconflicter

	// OpenAPI allOf discriminated types ("base types")
	bundleGroupSubTypes bool
	bundleBaseTypes     bool
}

// WithBundlePruneUnused instructs the analyzer to exclude unused schemas when preparing a bundled schema.
func WithBundlePruneUnused(prune bool) Option {
	return func(o *options) {
		o.pruneUnused = prune
	}
}

// WithBundleNameProvider equips the analyzer with a [NameProvider] to construct named schemas when bundling.
func WithBundleNameProvider(provider NameProvider) Option {
	return func(o *options) {
		o.bundleNameProvider = provider
	}
}

// WithBundlePathProvider equips the analyzer with a [PathProvider] to construct a path hierarchy in the bundled schema.
func WithBundlePathProvider(provider PackageNameProvider) Option {
	return func(o *options) {
		o.bundlePathProvider = provider
	}
}

// WithBundleStragegy defines the bundling strategy to adopt.
func WithBundleStragegy(strategy bundle.SchemaBundlingStragegy) Option {
	return func(o *options) {
		o.bundleStrategy = strategy
	}
}

// WithBundleAggressiveness ...
func WithBundleAggressiveness(aggressiveness bundle.SchemaBundlingAggressiveness) Option {
	return func(o *options) {
		o.bundleAggressiveness = aggressiveness
	}
}

// WithBundleEnumsPackage instructs [SchemaAnalyzer.Bundle] to move schemas with only enum validations into a
// subpackage of their parent.
//
// Example:
//
// The schema:
//
//	parent:
//	  type: { type of parent }
//	  enum:
//	    - value1
//	    - value2
//	    - value3
//
// Is refactored into:
//
//	parent:
//	  { ... }
//		allOf:
//		  - $ref: '#/{pkg}/parentEnum'
//
//	$defs:
//	  {pkg}:
//		  parentEnum:
//	      type: { type of parent }
//		    description: '{parentEnum} enumerates the valid values for a {parent}'
//			  enum:
//			 	  - value1
//				  - value2
//			 	  - value3
func WithBundleEnumsPackage(pkg string) Option {
	return func(o *options) {
		if pkg != "" {
			o.bundleEnumsPackage = pkg
		}
	}
}

// WithBundleNameIdentifier sets a unique identifier generator for schema names.
//
// The default is to just return the name.
func WithBundleNameIdentifier(unique UniqueIdentifier) Option {
	return func(o *options) {
		o.bundleNameIdentifier = unique
	}
}

func WithBundleNameDeconflicter(deconflicter Deconflicter) Option {
	return func(o *options) {
		o.bundleNameDeconflicter = deconflicter
	}
}

// WithBundlePathIdentifier sets a unique identifier generator for package paths.
//
// The default is to just return the path.
func WithBundlePathIdentifier(unique UniqueIdentifier) Option {
	return func(o *options) {
		o.bundlePathIdentifier = unique
	}
}

func WithBundlePathDeconflicter(deconflicter Deconflicter) Option {
	return func(o *options) {
		o.bundlePathDeconflicter = deconflicter
	}
}

// WithBundleSingleRoot ensures that the bundle ends up with a single top-most root level schema (possibly fictious),
// otherwise, several roots may be defined.
func WithBundleSingleRoot(enabled bool) Option {
	return func(o *options) {
		o.bundleSingleRoot = enabled
	}
}

// WithBundleBaseTypes discover all base types and form a subpackage.
func WithBundleBaseTypes(enabled bool) Option {
	return func(o *options) {
		o.bundleBaseTypes = enabled
	}
}

// WithBundleGroupSubTypes discover all subtypes of a given base type and regroup them in the same package as a the parent base type.
func WithBundleGroupSubTypes(enabled bool) Option {
	return func(o *options) {
		o.bundleGroupSubTypes = enabled
	}
}

// WithBacktrackOnConflicts enables deconfliction callbacks to backtrack on past naming decisions.
//
// Whenever enabled, the [Namespace] passed to deconfliction callbacks implements [BacktrackableNamespace],
// so the callback has the ability to force the [SchemaAnalyzer] to rename an object.
func WithBacktrackOnConflicts(enabled bool) Option {
	return func(o *options) {
		o.withBacktrack = enabled
	}
}
