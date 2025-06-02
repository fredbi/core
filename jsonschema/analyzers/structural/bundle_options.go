package structural

import "github.com/fredbi/core/jsonschema/analyzers/structural/bundle"

// BundleOption customizes the behavior of [SchemaAnalyzer.Bundle].
type BundleOption func(*bundleOptions)

type bundleOptions struct {
	PruneUnused          bool
	BundleStrategy       bundle.SchemaBundlingStragegy
	BundleAggressiveness bundle.SchemaBundlingAggressiveness
	BundleNameProvider   NameProvider
	BundlePathProvider   NameProvider
	BundleEnums          bool
	BundleEnumsPackage   string
	BundleNameEqual      EqualOperator
	BundlePathEqual      EqualOperator
	BundleMarker         SchemaMarker
	BundleSingleRoot     bool
}

// WithBundlePruneUnused instructs the analyzer to exclude unused schemas when preparing a bundled schema.
func WithBundlePruneUnused(prune bool) BundleOption {
	return func(o *bundleOptions) {
		o.PruneUnused = prune
	}
}

// WithBundleNameProvider equips the analyzer with a [NameProvider] to construct named schemas when bundling.
func WithBundleNameProvider(provider NameProvider) BundleOption {
	return func(o *bundleOptions) {
		o.BundleNameProvider = provider
	}
}

// WithBundlePathProvider equips the analyzer with a [PathProvider] to construct a path hierarchy in the bundled schema.
func WithBundlePathProvider(provider NameProvider) BundleOption {
	return func(o *bundleOptions) {
		o.BundlePathProvider = provider
	}
}

// WithBundleStragegy defines the bundling strategy to adopt.
func WithBundleStragegy(strategy bundle.SchemaBundlingStragegy) BundleOption {
	return func(o *bundleOptions) {
		o.BundleStrategy = strategy
	}
}

// WithBundleAggressiveness ...
func WithBundleAggressiveness(aggressiveness bundle.SchemaBundlingAggressiveness) BundleOption {
	return func(o *bundleOptions) {
		o.BundleAggressiveness = aggressiveness
	}
}

func WithBundleEnums(enabled bool) BundleOption {
	return func(o *bundleOptions) {
		o.BundleEnums = enabled
	}
}

func WithBundleEnumsPackage(pkg string) BundleOption {
	return func(o *bundleOptions) {
		if pkg != "" {
			o.BundleEnumsPackage = pkg
		}
	}
}

// WithBundleMarker sets a "mark", that is, injects new extensions before the schema is actually renamed or rebased.
func WithBundleMarker(marker SchemaMarker) BundleOption {
	return func(o *bundleOptions) {
		o.BundleMarker = marker
	}
}

func WithBundleNameEqualOperator(equal EqualOperator) BundleOption {
	return func(o *bundleOptions) {
		o.BundleNameEqual = equal
	}
}

func WithBundlePathEqualOperator(equal EqualOperator) BundleOption {
	return func(o *bundleOptions) {
		o.BundlePathEqual = equal
	}
}

// WithBundleSingleRoot ensures that the bundle ends up with a single top-most root level schema (possibly fictious),
// otherwise, several roots may be defined.
func WithBundleSingleRoot(enabled bool) BundleOption {
	return func(o *bundleOptions) {
		o.BundleSingleRoot = enabled
	}
}
