package models

import (
	"embed"
	"fmt"

	"github.com/fredbi/core/codegen/genapp"
	repo "github.com/fredbi/core/codegen/templates-repo"
	"github.com/fredbi/core/genmodels/generators/golang-models/providers"
	"github.com/fredbi/core/json/stores"
	store "github.com/fredbi/core/json/stores/default-store"
	"github.com/fredbi/core/jsonschema"
	"github.com/spf13/afero"
)

//go:embed templates
var embedFS embed.FS

// Option to configure the generation of models.
//
// This overrides any presets from a configuration file.
type Option func(*options)

type options struct {
	GenOptions

	store             stores.Store
	sourceSchemas     []string
	sourceOverlays    []string
	inputSchemas      jsonschema.Collection
	overlaySchemas    jsonschema.OverlayCollection // load optional overlays to merge on top of the load schemas
	innerOptions      []genapp.Option
	baseFS            afero.Fs
	outputPath        string // output folder
	overlayTemplates  []string
	wantsDumpAnalyzed bool
	namingOptions     []providers.Option

	// TODO: templates override
}

func (o options) validateOptions() error {
	if len(o.sourceSchemas) == 0 {
		return fmt.Errorf("the model generator requires at least one source to load schema. Use WithSourceSchemas(): %w", ErrInit)
	}
	if o.TargetImportPath == "" {
		return fmt.Errorf("the model generator requires an output folder to be specified. Use configuration or WithOutputPath(): %w", ErrInit)
	}

	return nil
}

func (o options) genappDefaults() []genapp.Option {
	return []genapp.Option{
		genapp.WithOutputAferoFS(o.baseFS),
		genapp.WithOutputPath(o.TargetImportPath),
		genapp.WithSkipFmt(o.SkipFmt),
		genapp.WithSkipCheckImport(o.SkipCheckImport),
		genapp.WithTemplates(
			embedFS, // TODO: allows overlay templates
			repo.WithAllowOverride(o.AllowTemplateOverride),
			repo.WithDumpTemplate(o.DumpTemplateFormat),
		),
	}
}

func (o options) genappOptionsWithDefaults() []genapp.Option {
	genappOptions := o.genappDefaults()
	genappOptions = append(genappOptions, o.innerOptions...)

	return genappOptions
}

func optionsWithDefaults(opts []Option) options {
	var o options

	for _, apply := range opts {
		apply(&o)
	}

	if o.store == nil {
		o.store = store.New()
	}

	if o.baseFS == nil {
		o.baseFS = afero.NewOsFs()
	}

	if o.inputSchemas.Len() == 0 {
		o.inputSchemas = jsonschema.MakeCollection(len(o.sourceSchemas), jsonschema.WithStore(o.store))
	}

	if o.overlaySchemas.Len() == 0 {
		o.overlaySchemas = jsonschema.MakeOverlayCollection(len(o.sourceOverlays), jsonschema.WithStore(o.store))
	}

	return o
}

// WithAferoFS overrides the default output to the os file system.
//
// This is primarily intended for testing purpose.
//
// Use this option with care, as the go imports check would need access to the go modules tree when resolving imports.
func WithAferoFS(fs afero.Fs) Option {
	return func(o *options) {
		if fs != nil {
			o.baseFS = fs
		}
	}
}

// WithOutputPath overrides the TargetImportPath setting to define the output folder.
//
// This overrides the TargetImportPath config setting.
func WithOutputPath(path string) Option {
	return func(o *options) {
		if path != "" {
			o.outputPath = path
		}
	}
}

/*
func WithOverlayTemplates(overlays fs.FS) Option {
	// TODO?
}
*/

// WithSourceSchemas specifies the locations (URLs or local files) from where to fetch
// the input schemas.
func WithSourceSchemas(locations ...string) Option {
	return func(o *options) {
		if locations != nil {
			o.sourceSchemas = locations
		}
	}
}

// WithSchemaCollection sets the input schema collection from which to generate models.
//
// This overrides [WithSourceSchemas].
func WithSchemaCollection(collection jsonschema.Collection) Option {
	return func(o *options) {
		o.inputSchemas = collection
	}
}

// WithSourceOverlays specifies the locations (URLs or local files) from where to fetch
// schema overlays.
func WithSourceOverlays(locations ...string) Option {
	return func(o *options) {
		if locations != nil {
			o.sourceOverlays = locations
		}
	}
}

// WithOverlayCollection sets the schema overlays to apply to input schemas.
//
// This overrides [WithSourceOverlays].
func WithOverlayCollection(collection jsonschema.OverlayCollection) Option {
	return func(o *options) {
		o.overlaySchemas = collection
	}
}

// WithGenAppOptions allows to customize the inner [genapp.GenApp] helper,
// so you may customize template funcmaps etc.
func WithGenAppOptions(opts ...genapp.Option) Option {
	return func(o *options) {
		o.innerOptions = append(o.innerOptions, opts...)
	}
}

// WithStore allows to customize the inner [stores.Store] used to hold JSON documents.
//
// By default, the default [store.Store] is used.
func WithStore(s stores.Store) Option {
	return func(o *options) {
		o.store = s
	}
}

// WithNameProviderOptions sets optional settings for the inner name provider.
func WithNameProviderOptions(opts ...providers.Option) Option {
	return func(o *options) {
		o.namingOptions = opts
	}
}
