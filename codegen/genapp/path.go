package genapp

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/fredbi/core/swag/mangling"
	"golang.org/x/tools/go/packages"
)

// PackagePath returns the go package path that corresponds to the output dir.
//
// This is the base import path for all packages generated there.
//
// PackagePath abides by go build rules to resolve a package name, and is go module-aware.
//
// The output path may not exist yet, or if it exists, it should not contain any go code, or a package name conflict
// may arise.
//
// # Module path
//
// If a module path is specified with the [WithModulePath] option, the package path will be moduleRoot followed by the last significant part in the
// output path.
//
// Example:
//
// With outputPath = "/tmp/go-generated/v2"
//
//	PackagePath("goswagger.io/go-openapi") => "goswagger.io/go-openapi/go-generated/v2"
//
// If the module root is empty, the resolver will resort to your local go build tree,
// either relying either on GOPATH or on the first parent go module.
//
// Example:
//
// Building from "$HOME/src/github.com/fredbi/core/genapp"
// with outputpath = "./go-generated/v2"
//
//	PackagePath("") => "github.com/fredbi/core/genapp/go-generated/v2"
//
// # Packages outside of the go build tree
//
// If you want to build outside of the build tree, the module root is required: use [WithModulePath].
// In that case, you will need to create a go module there with the returned package path
// before code generation is started.
//
// Notice that the calling process should have a write access to the output path,
// as [GoGenApp.PackagePath] simulates the go build process by temporarily creating the folder
// if it is missing.
func (g *GoGenApp) PackagePath(opts ...ModOption) (string, error) {
	modOpts := modOptionsWithDefaults(opts)

	return packagePath(g.outputPath, modOpts)
}

func packagePath(outputPath string, modOpts modOptions) (string, error) {
	pth, err := filepath.Abs(outputPath)
	if err != nil {
		return "", errors.Join(err, ErrGenApp)
	}

	// if the target doesn't exist yet, create the folder temporarily, so we may build from within.
	if _, fileExistsErr := os.Stat(pth); fileExistsErr != nil {
		if !os.IsNotExist(fileExistsErr) {
			return "", errors.Join(fileExistsErr, ErrGenApp)
		}

		if mkDirErr := os.MkdirAll(pth, 0755); mkDirErr != nil {
			return "", fmt.Errorf(
				"the calling process should have write access to %q to simulate a go build, or have %q already created: %w: %w",
				pth, outputPath, mkDirErr, ErrGenApp,
			)
		}

		defer func() {
			_ = os.Remove(pth)
		}()
	}

	mangler := mangling.New()
	shortPackage := mangler.ToGoPackageName(pth) // TODO: finish mangler to extract package short name

	// simulate a package file
	overlays := map[string][]byte{
		filepath.Join(pth, "doc.go"): []byte("package " + shortPackage + "\n"),
	}
	const pattern = "."

	moduleRoot := modOpts.modulePath // already sanitized
	if moduleRoot != "" {
		moduleName := path.Join(moduleRoot, shortPackage)

		// simulate a go mod file
		overlays[filepath.Join(pth, "go.mod")] = []byte("module " + moduleName + "\n")
	}

	config := &packages.Config{
		Mode:    packages.LoadImports,
		Dir:     pth,
		Overlay: overlays,
	}

	pkgs, err := packages.Load(config, pattern)
	if err != nil {
		return "", errors.Join(err, ErrGenApp)
	}

	if len(pkgs) == 0 {
		return "", fmt.Errorf(
			"could not resolve package %q load from %q (%q): %w",
			shortPackage, outputPath, pth, ErrGenApp,
		)
	}

	if len(pkgs) > 1 {
		return "", fmt.Errorf(
			"packages.Load resolved an ambiguous path for %q in %q: %w",
			shortPackage, pth, ErrGenApp,
		)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		errs := make([]error, 0, len(pkg.Errors))
		for _, err := range pkg.Errors {
			errs = append(errs, err)
		}
		errs = append(errs, ErrGenApp)

		return "", fmt.Errorf(
			"packages.Load resolved with errors: %w",
			errors.Join(errs...),
		)
	}

	return pkg.PkgPath, nil
}
