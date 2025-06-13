package models

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/fredbi/core/codegen/genapp"
	repo "github.com/fredbi/core/codegen/templates-repo"
	model "github.com/fredbi/core/genmodels/generators/golang-models/data-model"
	"github.com/fredbi/core/genmodels/generators/golang-models/providers"
	"github.com/fredbi/core/genmodels/generators/internal/log"
	"github.com/fredbi/core/json/stores"
	store "github.com/fredbi/core/json/stores/default-store"
	"github.com/fredbi/core/jsonschema"
	"github.com/fredbi/core/swag/loading"
	"github.com/spf13/afero"
)

//go:embed templates
var embedFS embed.FS

// Option to configure the generation of models.
//
// This overrides any presets from a configuration file.
type Option func(*options)

type options struct {
	model.GenOptions

	store             stores.Store
	sourceSchemas     []string
	sourceOverlays    []string
	inputSchemas      jsonschema.Collection
	overlaySchemas    jsonschema.OverlayCollection // load optional overlays to merge on top of the load schemas
	generatorOptions  []genapp.Option
	baseFS            afero.Fs // TODO: use fs.FS
	outputPath        string   // output folder
	overlayTemplates  []string
	wantsDumpAnalyzed bool
	namingOptions     []providers.Option
	dumpOutput        io.Writer
	templateOverlays  []fs.FS
	loadOptions       []loading.Option
	l                 log.Logger
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
		genapp.WithOutputPath(o.TargetImportPath),                // the target location
		genapp.WithSkipFormat(o.SkipFmt),                         // skip go fmt step (e.g. for debug)
		genapp.WithSkipCheckImport(o.SkipCheckImport),            // skip go import step (e.g. for debug)
		genapp.WithFormatGroupPrefixes(o.FormatGroupPrefixes...), /// specify imports grouping patterns
		// configure the templates repo
		genapp.WithTemplatesRepoOptions(
			repo.WithDumpTemplate(o.DumpTemplateFormat),
			repo.WithOverlays(o.templateOverlays...),
		),
	}
}

func (o options) genappOptionsWithDefaults() []genapp.Option {
	genappOptions := o.genappDefaults()
	genappOptions = append(genappOptions, o.generatorOptions...)

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
		o.baseFS = afero.NewOsFs() // TODO: use fs.FS
	}

	if o.inputSchemas.Len() == 0 {
		o.inputSchemas = jsonschema.MakeCollection(len(o.sourceSchemas), jsonschema.WithStore(o.store))
	}

	if o.overlaySchemas.Len() == 0 {
		o.overlaySchemas = jsonschema.MakeOverlayCollection(len(o.sourceOverlays), jsonschema.WithStore(o.store))
	}

	if o.l == nil {
		o.l = log.NewColoredLogger(log.WithName("golang-models"))
	}

	if o.dumpOutput == nil {
		o.dumpOutput = os.Stdout
	}

	return o
}

// WithAferoFS overrides the default output to the os file system.
//
// This is primarily intended for testing purpose.
//
// Use this option with care, as the go imports check would need access to the go modules tree when resolving imports.
//
// Also since we generate go.mod file using the go command, it is not possible to generate a go module within an [afero.Fs].
//
// TODO: this is sufficiently complicated like that, we could remove this option.
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

// WithTemplateOverlays allows for overriding the provided embeded templates with
// a collection of [fs.FS] containing alternate templates.
//
// Notice that the templates structure in the provided file systems must match exactly the
// original structure, as this setting does not allow prefix stripping rules.
//
// Defining overlays here automatically sets the option [GenOptions.AllowTemplateOverride].
func WithTemplateOverlays(overlays ...fs.FS) Option {
	return func(o *options) {
		o.templateOverlays = overlays
	}
}

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

// WithGenAppOptions allows to customize the generator [genapp.GenApp] helper,
// so you may customize template funcmaps etc.
func WithGenAppOptions(opts ...genapp.Option) Option {
	return func(o *options) {
		o.generatorOptions = append(o.generatorOptions, opts...)
	}
}

// WithGenOptions presets all generation settings defined by [model.GenOptions].
func WithGenOptions(genOptions model.GenOptions) Option {
	return func(o *options) {
		o.GenOptions = genOptions
	}
}

// WithStore allows to customize the generator [stores.Store] used to hold JSON documents.
//
// By default, the default [store.Store] is used.
func WithStore(s stores.Store) Option {
	return func(o *options) {
		o.store = s
	}
}

// WithNameProviderOptions sets optional settings for the generator name provider.
func WithNameProviderOptions(opts ...providers.Option) Option {
	return func(o *options) {
		o.namingOptions = opts
	}
}

// WithLogger overrides the default logger used during generation.
func WithLogger(l log.Logger) Option {
	return func(o *options) {
		o.l = l
	}
}

// WithDumpOutput sets the output for dumping template or analyzed schemas (for debug or documentation).
//
// The default is [os.Stdout].
func WithDumpOutput(w io.Writer) Option {
	return func(o *options) {
		o.dumpOutput = w
	}
}

// WithLoadOptions defines [loading.Option] s for the schema and overlay loaders from file or HTTP.
//
// This setting allows to tune the loader, for instance to support unrecognized file extensions or
// to load JSON schema assets from a [fs.FS] (e.g. [embed.FS]).
func WithLoadOptions(loadOptions ...loading.Option) Option {
	return func(o *options) {
		o.loadOptions = loadOptions
	}
}
