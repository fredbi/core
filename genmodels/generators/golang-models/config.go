package models

import (
	"errors"
	"fmt"

	"github.com/fredbi/core/codegen/genapp"
	settings "github.com/fredbi/core/genmodels/generators/common-settings"
	"github.com/fredbi/core/genmodels/generators/golang-models/providers"
	"github.com/fredbi/core/genmodels/generators/golang-models/schema"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
	"github.com/spf13/afero"
)

func (g *Generator) init() error {
	// load the configuration from defaults, configuration file or CLI flags (applied in this order)
	if err := g.loadConfig(); err != nil {
		return err
	}

	if g.l.DebugEnabled() {
		if err := g.dumpSettings(); err != nil {
			return err
		}
	}

	// initialize the [genapp.GoGenApp] with templates, etc.
	g.generator = genapp.New(embedFS, g.genappOptionsWithDefaults()...)

	// initialize the target directory, fail early if there is an issue there
	if err := g.initOutput(); err != nil {
		return err
	}

	// initialize the name provider
	namingOptions := []providers.Option{
		providers.WithBaseImportPath(g.BaseImportPath), // the nameProvider needs the base import path to generate fully qualified package names
	}
	namingOptions = append(namingOptions, g.namingOptions...)
	g.nameProvider = providers.NewNameProvider(namingOptions...)

	// initialize the schema builder
	builder := schema.NewBuilder(schema.WithNameProvider(g.nameProvider)) // builders leverage the name provider for generation based on enum value
	g.schemaBuilder = builder                                             // [schema.Builder] implements both
	g.packageBuilder = builder

	// configure options for the analyzer
	analyzerOptions := []structural.Option{
		// validates all supported extensions
		structural.WithExtensionMappers(
			g.nameProvider.MapExtension,  // all extensions that affect naming and layout
			g.schemaBuilder.MapExtension, // all extensions that affect type mapping
		),
		// combo option to refactor JSON schema constructs
		structural.WithRefactorSchemas(true),
		// Name provider for refactored named schemas
		// this general-purpose name provider method inspects [structural.AnalyzedSchema.IsRefectored]
		structural.WithSplitOverlappingAllOf(true),
		structural.WithRefactorEnums(g.PackageLayoutMode.Has(settings.PackageLayoutEnums)), // allow enums to be defined in their own package
	}

	// configure options for Bundle
	bundleOptions, err := g.makeBundleOptions()
	if err != nil {
		return err
	}

	analyzerOptions = append(analyzerOptions, bundleOptions...)

	g.analyzer = structural.NewAnalyzer(analyzerOptions...)

	// last minute injection of audit callbacks to the name provider
	if g.WantsAudit {
		g.nameProvider.SetAuditor(g.analyzer)
	}
	g.nameProvider.SetMarker(g.analyzer)

	if g.WantsGenMetadata {
		g.nameProvider.SetAnnotator(g.analyzer)
	}

	return nil
}

// loadConfig loads and merge the codegen config from file, flags or options passed to [New] (in this order).
//
// TODO
func (g *Generator) loadConfig() error {
	// load config file

	// apply CLI flags

	// apply options in new

	return g.validateOptions()
}

func (g *Generator) initOutput() error {
	if err := g.ensureOutputPath(); err != nil {
		return errors.Join(err, ErrModel)
	}

	// g.outputPath is now guaranteed to exist, be empty and user-writable

	// check if go.mod needs to be initialized
	isGoModRequired, err := g.generator.IsGoModRequired()
	if err != nil {
		return errors.Join(err, ErrModel)
	}

	if isGoModRequired {
		if g.TargetModuleRoot == "" {
			return fmt.Errorf(
				"target %q is outside the go build tree and a go.mod file is required. Please specify a TargetModuleRoot: %w",
				g.outputPath, ErrModel,
			)
		}

		if !g.WantsGoMod {
			g.l.Warn(
				"a go.mod file is required and will be generated, even though this requirement is missing from configuration",
				"target_dir", g.outputPath,
				"hint", "if you don't want to see this warning again, enable WantsGoMod (go.mod generation) in the config",
			)

			g.WantsGoMod = true
		}
	}

	// The module path provided here is the parent. The package name is extracted from
	// the base part of the outputPath. See [genapp.GoGenApp.PackagePath].
	//
	// Hence:
	//
	// if
	//   TargetModuleRoot = goswagger.io/go-openapi
	// and
	//   targetOutput = ./generated/models
	//
	// The resulting package will be:
	//
	//   goswagger.io/go-openapi/models
	pkgName, err := g.generator.PackagePath(genapp.WithModulePath(g.TargetModuleRoot))
	if err != nil {
		return fmt.Errorf(
			"an error occured when checking for a valid package fully qualified name in %q",
			g.outputPath,
		)

	}

	g.BaseImportPath = pkgName

	if g.WantsGoMod {
		// prepare go mod
		if fsName := g.baseFS.Name(); fsName != "OsFs" {
			g.l.Warn(
				"module initialization likely won't work with a virtual file system",
				"fs_type", fsName,
				"hint", "virtual file system (afero) is recommended only when running tests",
			)
		}

		if err := g.generator.GoMod(
			genapp.WithModulePath(pkgName),       // this is the fully qualified package name, within TargetModule root if defined, or as resolved from the current go build tree
			genapp.WithGoVersion(g.MinGoVersion), // if empty, will apply go mod defaults, i.e. latest install go version
		); err != nil {
			return errors.Join(err, ErrModel)
		}

		g.l.Info(
			"go module initialized",
			"target_package", g.BaseImportPath,
			"target_dir", g.outputPath,
		)
	}

	return nil
}

func (g *Generator) ensureOutputPath() (err error) {
	var outputDirExists, outputDirIsEmpty bool

	outputDirExists, err = afero.DirExists(g.baseFS, g.outputPath)
	if err != nil {
		return err
	}

	if outputDirExists {
		outputDirIsEmpty, err = afero.IsEmpty(g.baseFS, g.outputPath)
		if err != nil {
			return err
		}

		if !outputDirIsEmpty {
			g.l.Error(
				"target output error",
				"target_dir", g.outputPath,
				"hint", "the target may contain content from a previous generation. Please make sure the target is fresh.",
			)

			return fmt.Errorf(
				"target %q already exists and is not empty. Please make sure the target is empty before launching the generation",
				g.outputPath,
			)
		}

		if !isUserWritableDir(g.baseFS, g.outputPath) {
			return fmt.Errorf(
				"target %q already exists and is not writable. Please make sure the target is writable before launching the generation",
				g.outputPath,
			)
		}

		return nil
	}

	if err = g.baseFS.MkdirAll(g.outputPath, 0755); err != nil {
		return err
	}

	return nil
}

func isUserWritableDir(baseFS afero.Fs, dir string) bool {
	file, err := afero.TempFile(baseFS, dir, "_check_writable_")
	if err != nil {
		return false
	}

	_ = file.Close()
	_ = baseFS.Remove(file.Name())

	return true
}
