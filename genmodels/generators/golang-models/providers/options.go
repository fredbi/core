package providers

import (
	"github.com/fredbi/core/genmodels/generators/golang-models/ifaces"
	"github.com/fredbi/core/mangling"
)

type Option func(o *options)

type options struct {
	mangler         ifaces.NameMangler
	manglingOptions []mangling.Option
	auditor         ifaces.Auditor
	marker          ifaces.Marker
	baseImportPath  string
}

// SetAuditor does the same as [WithAuditor], but after the [NameMangler] has been already initialized.
func (o *options) SetAuditor(auditor ifaces.Auditor) {
	o.auditor = auditor
}

// SetMarker does the same as [WithMarker], but after the [NameMangler] has been already initialized.
func (o *options) SetMarker(marker ifaces.Marker) {
	o.marker = marker
}

func optionsWithDefaults(opts []Option) options {
	o := options{}

	for _, apply := range opts {
		apply(&o)
	}

	return o
}

// WithMangler injects a custom name mangler.
//
// The default mangle configured is [mangling.NameMangler].
func WithMangler(mangler ifaces.NameMangler) Option {
	return func(o *options) {
		o.mangler = mangler
	}
}

// WithManglerOptions inject [mangling.Option] s to customize the behavior of the inner [mangling.NameMangler]
//
// This is disabled when using a custom mangler injected by [WithMangler].
func WithManglerOptions(opts ...mangling.Option) Option {
	return func(o *options) {
		o.manglingOptions = opts
	}
}

// WithAuditor injects an [ifaces.Auditor] to track the decisions of the [NameProvider]
//
// If the [Auditor] is not defined at initialization time, you may set it later using SetAuditor().
func WithAuditor(auditor ifaces.Auditor) Option {
	return func(o *options) {
		o.auditor = auditor
	}
}

// WithMarker injects a [ifaces.Marker] to inject schema extensions from the [NameProvider]
//
// If the [Marker] is not defined at initialization time, you may set it later using SetMarker().
func WithMarker(marker ifaces.Marker) Option {
	return func(o *options) {
		o.marker = marker
	}
}

// WithBaseImportPath sets the base path to be joined to packages paths and form a fully qualified package names.
func WithBaseImportPath(path string) Option {
	return func(o *options) {
		o.baseImportPath = path
	}
}
