package jsonschema

import (
	"github.com/fredbi/core/json"
	"github.com/fredbi/core/jsonschema/overlay"
)

// Option to customize the behavior of a [Schema] document.
type Option func(*options)

type options struct {
	version         Version
	documentOptions []json.Option
	useDollarData   bool // support for $data ajv extension
}

// TODO: as usual, transform this to use pool
func optionsWithDefaults(opts []Option) *options {
	var o options

	for _, apply := range opts {
		apply(&o)
	}

	return &o
}

// WithVersion enforces a jsonschema dialect version.
func WithVersion(version Version) Option {
	return func(o *options) {
		o.version = version
	}
}

func WithDocumentOptions(opts ...json.Option) Option {
	return func(o *options) {
		o.documentOptions = append(o.documentOptions, opts...)
	}
}

// WithDollarData adds support for the "$data" extension popularized by the ajv validator.
//
// This extension applies to any jsonschema version or dialect.
func WithDollarData(enabled bool) Option {
	return func(o *options) {
		o.useDollarData = enabled
	}
}

func withOptions(opts *options) Option {
	return func(o *options) {
		*o = *opts
	}
}

// OverlayOption customizes the behavior of a schema [Overlay].
type OverlayOption func(*overlayOptions)

type overlayOptions struct {
	version         overlay.Version
	documentOptions []json.Option
}

func overlayOptionsWithDefaults(opts []OverlayOption) *overlayOptions {
	var o overlayOptions // TODO: borrow from pool

	for _, apply := range opts {
		apply(&o)
	}

	return &o
}

// WithOverlayVersion enforces a version of the overlay specification.
//
// At this moment only [overlay.VersionUndefined] and [overlay.Version10] are supported.
func WithOverlayVersion(version overlay.Version) OverlayOption {
	return func(o *overlayOptions) {
		o.version = version
	}
}

func WithOverlayDocumentOptions(opts ...json.Option) OverlayOption {
	return func(o *overlayOptions) {
		o.documentOptions = append(o.documentOptions, opts...)
	}
}

func withOverlayOptions(opts *overlayOptions) OverlayOption {
	return func(o *overlayOptions) {
		*o = *opts
	}
}
