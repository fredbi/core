package golang

import (
	"fmt"
	"io/fs"

	"github.com/fredbi/core/codegen/genapp"
	"github.com/spf13/afero"
)

type Option func(*options)

type options struct {
	GenOptions
	sourceSchemas  []string
	overlaySchemas map[string]string // load optional overlays to merge on top of the load schemas
	innerOptions   []genapp.Option
	baseFS         afero.Fs
	outputPath     string // output folder
	// TODO: templates override
	overlayTemplates []string
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

func optionsWithDefaults(opts []Option) options {
	var o options
	o.baseFS = afero.NewOsFs()

	for _, apply := range opts {
		apply(&o)
	}

	return o
}

// WithAferoFS overrides the default os.FS output, e.g. for testing purpose.
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

func WithOverlayTemplates(overlays fs.FS) Option {
	// TODO?
}

// WithGenAppOptions allows to customize the inner [genapp.GenApp] helper,
// so you may customize template funcmaps etc.
func WithGenAppOptions(opts ...genapp.Option) Option {
	return func(o *options) {

		o.innerOptions = append(o.innerOptions, opts...)
	}
}
