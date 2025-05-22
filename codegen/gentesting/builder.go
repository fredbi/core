package gentest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"runtime"
	"testing"
)

type Builder struct {
	options

	packageLocation    string
	pluginMainLocation string
	symbols            symbolsIndex
	name               string
	p                  *plugin.Plugin
}

// New test plugin builder to test go code dynamically.
//
// It assumes the provided package location is buildable by go (i.e. inside the GOROOT tree
// or with appropriate parent module declare).
//
// It panics if the packageLocation is empty.
func New(packageLocation string, _ ...Option) *Builder {
	g := &Builder{
		packageLocation: packageLocation,
	}

	// check settings
	if g.packageLocation == "" {
		panic("empty package location")
	}

	if g.pluginMainLocation == "" {
		g.pluginMainLocation = filepath.Join(
			filepath.Dir(g.packageLocation),
			filepath.Base(g.packageLocation)+"-testplugin", // TODO: randomize this, so as to allow parallel tests on same package
		)
		/*
			// building outside GOROOT requires a go mod
			tmpLocation, err := os.MkdirTemp("", filepath.Base(g.packageLocation)+"-testplugin")
			if err != nil {
				panic(fmt.Errorf("could not create temporary directory: %w", err))
			}
			g.pluginMainLocation = tmpLocation
		*/
	}

	return g
}

// Build a test plugin from generated source.
//
// The plugin object will be written to disk alongside the source package repository.
//
// The build process carries out the following tasks:
//
//   - asserts that the provided package compiles
//   - discover all exported symbols from the package
//   - builds a plugin shared library locally
//   - loads this plugin
//   - ensures that all exported symbols are visible to the plugin
func (g *Builder) Build() error {
	const pluginFile = "plugin.go"
	if err := os.MkdirAll(g.pluginMainLocation, 0755); err != nil {
		return err
	}

	pluginMainFile := filepath.Join(g.pluginMainLocation, pluginFile)
	if err := g.genRexporter(pluginMainFile); err != nil {
		return err
	}

	pluginObjectFile := g.name + ".so" // .so on linux how about other os'es?
	if err := g.buildPlugin(pluginFile, pluginObjectFile); err != nil {
		return err
	}

	p, err := plugin.Open(filepath.Join(g.pluginMainLocation, pluginObjectFile))
	if err != nil {
		return err
	}

	runtime.AddCleanup(p, cleanup, g.pluginMainLocation)

	g.p = p

	// sanity check all exported symbols
	for kind, list := range g.symbols {
		for _, element := range list {
			_, e := g.p.Lookup(element.Ident)
			if e != nil {
				return fmt.Errorf("internal error: plugin lookup for exported %v %q didn't resolve as expected", kind, element.Ident)
			}
		}
	}

	return nil
}

// Symbols of a given kind from the source package being tested.
func (g *Builder) Symbols(kind ExportedSymbolKind) []string {
	return g.symbols.NamesForKind(kind)
}

// Driver yields a test [Driver] helper object, which is ready to run tests.
func (g *Builder) Driver() *Driver {
	// construct indexes for the driver to use during lookups
	index := make(map[string]symbol)
	identKindIndex := make(map[ExportedSymbolKind]map[string]symbol)
	for _, kind := range []ExportedSymbolKind{
		SymbolConst,
		SymbolVar,
		SymbolType,
		SymbolFunc,
	} {
		identKindIndex[kind] = make(map[string]symbol)
	}

	for _, list := range g.symbols {
		for _, symb := range list {
			index[symb.Ident] = symb

			kindIndex := identKindIndex[symb.Kind]
			kindIndex[symb.Ident] = symb
			identKindIndex[symb.Kind] = kindIndex
		}
	}

	return &Driver{
		symbols:        g.symbols,
		p:              g.p,
		identIndex:     index,
		identKindIndex: identKindIndex,
	}
}

func (g *Builder) Asserter(t *testing.T) *Asserter {
	return g.Driver().Asserter(t)
}

// Plugin yields the inner [plugin.Plugin] object.
func (g *Builder) Plugin() *plugin.Plugin {
	return g.p
}

// Name yields the package name for which the plugin has been built
func (g *Builder) Name() string {
	return g.name
}

// Cleanup the generated test plugin.
func (g *Builder) Cleanup() {
	cleanup(g.pluginMainLocation)
}

// genRexporter constructs a main package plugin bootstrap which reexport all exported symbols from a package.
func (g *Builder) genRexporter(pluginMainFile string) error {
	// first step is to parse the code and discover all the exported symbols
	p := newParser(g.packageLocation)
	if err := p.Parse(); err != nil {
		return err
	}

	// populate internal index of exported symbols
	g.symbols = p.Symbols()
	g.name = p.PackageName()

	b := newBootStrapper(g.symbols, p.PackageImportPath(), p.PackageName(), pluginMainFile)

	// generate a bootstrap main for the plugin
	return b.Generate()
}

// buildPlugin compiles as a plugin the go code at the given location and injects some plugin wrapper.
func (g *Builder) buildPlugin(pluginMainFile, pluginObjectFile string) error {
	builder := exec.Command(
		"go",
		"build",
		"-buildmode=plugin",
		"-o", pluginObjectFile,
		pluginMainFile,
	)
	builder.Dir = g.pluginMainLocation
	builder.Stderr = os.Stderr
	builder.Stdout = os.Stdout

	return builder.Run()
}

func cleanup(pluginMainLocation string) {
	_ = os.RemoveAll(pluginMainLocation)
}
