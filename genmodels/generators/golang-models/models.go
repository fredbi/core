package models

import (
	"context"
	"runtime"
	stdsync "sync"

	"github.com/fredbi/core/codegen/genapp"
	"github.com/fredbi/core/genmodels/generators/golang-models/providers"
	"github.com/fredbi/core/genmodels/generators/internal/sync"
	"github.com/fredbi/core/jsonschema/analyzers"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
	"github.com/fredbi/core/jsonschema/analyzers/structural/order"
	"golang.org/x/sync/errgroup"
)

// Generator knows how to build go types from JSON schema models.
type Generator struct {
	options

	inner          *genapp.GoGenApp
	deconflicter   *providers.NameDeconflicter
	analyzer       structural.Analyzer
	nameProvider   *providers.NameProvider
	stashedSchemas map[analyzers.UniqueID]TargetModel
	stashMx        stdsync.RWMutex
}

// New generator to build go types.
func New(opts ...Option) *Generator {
	g := &Generator{
		options: optionsWithDefaults(opts),
	}

	g.nameProvider = providers.NewNameProvider(g.namingOptions...)
	g.stashedSchemas = make(map[analyzers.UniqueID]TargetModel)

	return g
}

// Generate the models.
func (g *Generator) Generate() error {
	// 1. configuration & input schema analysis
	if err := g.init(); err != nil {
		return err
	}

	if g.WantsDumpTemplates {
		// invoked for debug or doc update: dump templates and exit
		return g.dumpTemplates()
	}

	// 2. retrieve schemas from files or URLs
	if err := g.loadJSONSchemas(); err != nil {
		return err
	}

	// 3. apply overlays if any
	if err := g.loadOverlays(); err != nil {
		return err
	}

	// 4. analyze input schemas
	if err := g.analyze(); err != nil {
		return err
	}

	// At this stage, everything we need to know about JSON schemas is well understood by the analyzer.
	// Any invalid construct in the input should have been detected by now.

	// 5. plan ahead the future layout for generated package(s)
	//
	// Errors may be still be raised, e.g. if we don't find a way to deconflict the namespace or some internal error (bug).

	if err := g.planPackageLayout(); err != nil {
		return err
	}

	// At this stage, all things that are going to produce types are known,
	// with a unique package and name layout.
	if g.wantsDumpAnalyzed {
		return g.dumpAnalyzed()
	}

	// 6. generate go models
	// Errors may be i/o errors or some internal error (e.g. invalid template, invalid code can't be formatted...)
	return g.generate()
}

func (g *Generator) generate() error {
	var sem *sync.WatermarkSemaphore

	// prepare for concurrent generation, if allowed
	if g.maxParallel() > 1 {
		sem = sync.NewWatermarkSemaphore()
	}
	genGroup, ctx := errgroup.WithContext(context.Background())
	genGroup.SetLimit(g.maxParallel())
	schemaGroup, _ := errgroup.WithContext(ctx)
	schemaGroup.SetLimit(g.maxParallel())

	scopeIterator := g.analyzer.AnalyzedSchemas(
		// after the layout stage, everything we want with a name is named: let the genModel deal with the anonymous stuff.
		structural.OnlyNamedSchemas(),
		// walk up the dependency graph from leaves to roots.
		structural.WithOrderedSchemas(order.BottomUp),
		// skip schemas defined as placeholder containers to emulate a package: don't build a type from those
		structural.WithFilterFunc(func(s *structural.AnalyzedSchema) bool {
			return !s.HasExtension("x-go-namespace-only") // TODO: use settings to get aliases etc
		}),
	)

	// render all analyzed schemas, walking up the dependency graph
	for analyzedSchema := range scopeIterator {
		// parallel execution of the template rendering
		genGroup.Go(func() error {
			// wait until all dependencies are processed (or a rendering has failed). It's okay if sem is nil.
			if err := sem.Acquire(ctx, analyzedSchema.RequiredIndex); err != nil {
				return err
			}
			// all requirements have been processed: may go ahead in parallel

			const modelTemplate = "model" // we just need this top-level entry in the templates repository

			for genModel := range g.makeGenModels(analyzedSchema) {
				schemaGroup.Go(func() error {
					return g.inner.Render(modelTemplate, genModel)
				})
			}

			if err := schemaGroup.Wait(); err != nil {
				return err
			}

			sem.Release(analyzedSchema.Index)

			return nil
		})
	}

	return genGroup.Wait()
}

func (g *Generator) init() error {
	// load the configuration from defaults, configuration file or CLI flags (applied in this order)
	if err := g.loadConfig(); err != nil {
		return err
	}

	// initialize the [genapp.GoGenApp] with templates, etc.
	// deferring the configuration of the inner [GoGenApp] until the config is loaded
	// TODO: resolve g.TargetImportPath (in go-swagger, this is very complicated.
	g.inner = genapp.New(g.genappOptionsWithDefaults()...)

	return nil
}

// analyze the collection of input json schemas
func (g *Generator) analyze() error {
	g.analyzer = structural.NewAnalyzer(
		structural.WithExtensionMapper(g.nameProvider.MapExtension), // validates all supported extensions
	) // configure options for the analyzer

	return g.analyzer.AnalyzeCollection(g.inputSchemas)
}

// maxParallel establishes how many go routines we may spawn when generating in parallel.
//
// By default, parallel generation is enabled with runtime.GOMAXPROCS routines.
//
// Setting the MaxParallel option to a negative value or 1 disables parallel generation.
func (g *Generator) maxParallel() int {
	switch {
	case g.MaxParallel < 0:
		return 1
	case g.MaxParallel == 0:
		return runtime.GOMAXPROCS(-1)
	default:
		return g.MaxParallel
	}
}
