package models

import (
	"context"
	"runtime"
	stdsync "sync"

	"github.com/fredbi/core/codegen/genapp"
	model "github.com/fredbi/core/genmodels/generators/golang-models/data-model"
	"github.com/fredbi/core/genmodels/generators/golang-models/providers"
	"github.com/fredbi/core/genmodels/generators/golang-models/schema"
	"github.com/fredbi/core/genmodels/generators/internal/sync"
	"github.com/fredbi/core/jsonschema/analyzers"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
	"github.com/fredbi/core/jsonschema/analyzers/structural/order"
	"golang.org/x/sync/errgroup"
)

// Generator knows how to build go types from JSON schema models.
type Generator struct {
	options

	analyzer       structural.Analyzer
	generator      *genapp.GoGenApp
	deconflicter   *providers.NameDeconflicter
	nameProvider   NameProvider
	schemaBuilder  SchemaBuilder
	packageBuilder PackageBuilder

	stashedSchemas map[analyzers.UniqueID]model.TargetModel
	stashMx        stdsync.RWMutex
}

// New code generator to build go types.
func New(opts ...Option) *Generator {
	g := &Generator{
		options: optionsWithDefaults(opts),
	}

	g.nameProvider = providers.NewNameProvider(g.namingOptions...)
	g.stashedSchemas = make(map[analyzers.UniqueID]model.TargetModel)

	builder := schema.NewBuilder(schema.WithNameProvider(g.nameProvider)) // builders leverage the name provider for generation based on enum value
	g.schemaBuilder = builder                                             // [schema.Builder] implements both
	g.packageBuilder = builder

	// the rest of the initialization is deferred to after loading configuration.

	return g
}

// Generate the models.
//
// The generation process essentially consists of 4 steps:
//
//  1. analysis: a structural analysis of the JSON schema(s): raw JSON schema grammar and semantic subtleties are made more amenable
//  2. bundling: an internal reorganization of the schema(s) structure, to define a '$ref's (named entities) with the
//     appropriate strategy and rename things as expected to produce some go code.
//  3. layout: prepare target folders, possibly with some generated package-level artifact (e.g. doc.go)
//     (parallelizable)
//  4. models generation: iterate over all analyzed schemas, walking up the dependency graph, and generate source code
//     for go type definitions (parallelizable).
func (g *Generator) Generate() error {
	// 1. configuration & input schema analysis
	if err := g.init(); err != nil {
		g.l.Error("could not load config", "err", err)

		return err
	}
	g.l.Debug("model generator initialized")
	// TODO: on debug show settings

	if g.WantsDumpTemplates {
		// invoked for debug or doc update: dump templates and exit
		g.l.Debug("want templates dumped")

		return g.dumpTemplates()
	}

	// 2. retrieve schemas from files or URLs
	if err := g.loadJSONSchemas(); err != nil {
		g.l.Error("could not load schemas", "err", err)

		return err
	}
	g.l.Debug("schemas loaded", "schemas", g.inputSchemas.Len())

	// 3. apply overlays if any
	if err := g.loadOverlays(); err != nil {
		g.l.Error("could not load overlays", "err", err)

		return err
	}
	g.l.Debug("overlays applied", "overlays", g.overlaySchemas.Len())

	// 4. analyze input schemas
	if err := g.analyze(); err != nil {
		g.l.Error("could not analyze schemas", "err", err)

		return err
	}
	g.l.Info("schemas analyzed", "schemas", g.analyzer.Len())

	// At this stage, everything we need to know about JSON schemas is well understood by the analyzer.
	// Any invalid construct in the input should have been detected by now.

	// 5. plan ahead the future layout for generated package(s)
	//
	// Errors may be still be raised, e.g. if we don't find a way to deconflict the namespace or some internal error (bug).
	if err := g.planPackageLayout(); err != nil {
		g.l.Error("could not prepared layout", "err", err)

		return err
	}
	g.l.Info("package layout for models done")

	// At this stage, all things that are going to produce types are known,
	// with a unique package and name layout.
	if g.wantsDumpAnalyzed {
		g.l.Debug("want analyzer dumped")

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
	g.l.Info("starting models generation", "parallelism", g.maxParallel())

	scopeIterator := g.analyzer.AnalyzedSchemas(
		// after the layout stage, everything we want with a name is named: let the genModel deal with the anonymous stuff.
		structural.OnlyNamedSchemas(),
		// walk up the dependency graph from leaves to roots.
		structural.WithOrderedSchemas(order.BottomUp),
		// skip schemas defined as placeholder containers to emulate a package: don't build a type from those
		structural.WithFilterFunc(func(s *structural.AnalyzedSchema) bool {
			return !s.HasExtension("x-go-namespace-only")
		}),
	)

	// render all analyzed schemas, walking up the dependency graph

	// metrics
	var numSchemas int

	for analyzedSchema := range scopeIterator {
		// parallel execution of the template rendering
		numSchemas++

		genGroup.Go(func() error {
			// wait until all dependencies are processed (or a rendering has failed). It's okay if sem is nil.
			if err := sem.Acquire(ctx, analyzedSchema.RequiredIndex); err != nil {
				return err
			}
			// all the requirements have been processed: may go ahead in parallel

			for genModel := range g.makeGenModels(analyzedSchema) {
				modelTemplate := genModel.Template // e.g "model"

				schemaGroup.Go(func() error {
					return g.generator.Render(modelTemplate, genModel.FileName(), genModel)
				})
			}

			if err := schemaGroup.Wait(); err != nil {
				return err
			}

			sem.Release(analyzedSchema.Index)

			return nil
		})
	}

	err := genGroup.Wait()
	if err != nil {
		g.l.Error("could not complete models generation", "processed schemas", numSchemas, "err", err)
	}

	g.l.Info("models generation done", "processed schemas", numSchemas)

	return nil
}

func (g *Generator) init() error {
	// load the configuration from defaults, configuration file or CLI flags (applied in this order)
	if err := g.loadConfig(); err != nil {
		return err
	}

	// initialize the [genapp.GoGenApp] with templates, etc.
	// deferring the configuration of the generator [GoGenApp] until the config is loaded
	// TODO: resolve g.TargetImportPath (in go-swagger, this is very complicated.
	g.generator = genapp.New(embedFS, g.genappOptionsWithDefaults()...)

	return nil
}

// analyze the collection of input json schemas
func (g *Generator) analyze() error {
	g.analyzer = structural.NewAnalyzer(
		// configure options for the analyzer
		//
		// validates all supported extensions
		structural.WithExtensionMappers(
			g.nameProvider.MapExtension, // all extensions that affect naming and layout
			g.MapExtensionForType,       // all extensions that affect type mapping
		),
	)

	return g.analyzer.AnalyzeCollection(g.inputSchemas)
}

// maxParallel determines how many go routines may be spawned when generating in parallel.
//
// By default, parallel generation is enabled with runtime.GOMAXPROCS go routines.
//
// Setting the MaxParallel option to a negative value or to 1 disables parallel generation.
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
