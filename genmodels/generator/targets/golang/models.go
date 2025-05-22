package golang

import (
	"os"

	"github.com/fredbi/core/codegen/genapp"
	repo "github.com/fredbi/core/codegen/templates-repo"
	"github.com/fredbi/core/jsonschema"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

// Generator knows how to build go types from JSON schema models.
type Generator struct {
	options

	inner        *genapp.GoGenApp
	inputSchemas jsonschema.SchemaCollection
	analyzer     *structural.Analyzer

	locations LocationIndex // package planning index
	// TODO: index of stacked anonymous schemas (separate type, concurrent-safe)
}

// New generator to build go types.
func New(opts ...Option) *Generator {
	g := &Generator{
		options: optionsWithDefaults(opts),
	}

	return g
}

// Generate the models.
func (g *Generator) Generate() error {
	// configuration & input schema analysis
	if err := g.buildup(); err != nil {
		return err
	}

	if g.WantsDumpTemplates {
		// invoked for debug or doc update: dump templates and exit
		g.inner.Templates().Dump(os.Stdout)

		return nil
	}

	if err := g.loadJSONSchemas(); err != nil {
		return err
	}

	if err := g.analyze(); err != nil {
		return err
	}

	// plan ahead the future layout for generated package(s)
	if err := g.planPackageLayout(); err != nil {
		return err
	}

	// render all analyzed schemas, walking up the dependency graph
	// we just need one entry in the templates repository
	const modelTemplate = "model"

	for analyzedSchema := range g.analyzer.BottomUpSchemas() {
		for _, genSchema := range g.makeGenSchemas(analyzedSchema) {
			if err := g.inner.Render(modelTemplate, genSchema); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Generator) buildup() error {
	if err := g.loadConfig(); err != nil {
		return err
	}

	// deferring the configuration of the inner [GoGenApp] until the config is loaded
	// TODO: resolve g.TargetImportPath (in go-swagger, this is very complicated.
	defaultOptions := []genapp.Option{
		genapp.WithAferoFS(g.baseFS),
		genapp.WithOutputPath(g.TargetImportPath),
		genapp.WithSkipFmt(g.SkipFmt),
		genapp.WithSkipCheckImport(g.SkipCheckImport),
		genapp.WithTemplates(
			embedFS, // TODO: allows overlay templates
			repo.WithAllowOverride(g.AllowTemplateOverride),
			repo.WithDumpTemplate(g.DumpTemplateFormat),
		),
	}
	defaultOptions = append(defaultOptions, g.innerOptions...)
	g.inner = genapp.New(defaultOptions...)

	return nil
}

// loadConfig loads and merge the codegen config from file, flags or embedded options
func (g *Generator) loadConfig() error {
	// TODO: check fs vs g.TargetImportPath
	// options overrides
	if g.outputPath != "" {
		g.TargetImportPath = g.outputPath
	}

	// TODO fs

	return g.validateOptions()
}

// loadJSONSchemas loads JSON schema definitions from file or URL.
//
// Applies configured overlays if any.
func (g *Generator) loadJSONSchemas() error {
	return nil
}

// analyze the collection of input json schemas
func (g *Generator) analyze() error {
	g.analyzer = structural.New() // configure options for the analyzer

	if err := g.analyzer.AnalyzeCollection(g.inputSchemas); err != nil {
		return nil
	}

	return nil
}
