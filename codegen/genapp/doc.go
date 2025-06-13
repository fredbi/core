// Package genapp exposes [GoGenApp], a general-purpose golang code generation helper.
//
// [GoGenApp] stitches together a templates [repo.Repository] and knows how to render generated go files.
//
// [GoGenApp] may be used to initialize and update a go.mod file for the generated package.
//
// NOTE: using the imports grouping option affects the global setting [imports.LocalPrefix],
// and may cause side effects to other components using [imports.Process].
//
// # Dependencies
//
// This package is exposed as an independent module and imposes few dependencies.
//
// It uses internally github.com/spf13/afero as an abstraction of a writable file system.
//
// Code formatting relies on golang.org/x/tools/imports
//
// Excluding test dependencies, it relies directly on the following github.com/fredbi/core packages:
//
//   - github.com/fredbi/core/codegen/funcmaps
//   - github.com/fredbi/core/codegen/templates-repo
//
// And indirectly on:
//
//   - github.com/fredbi/core/swag/mangling
//   - github.com/fredbi/core/swag/stringutils
//
// Other dependencies found are test dependencies.
package genapp
