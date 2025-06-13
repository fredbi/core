// Package repo exposes a [Repository] to cache templates.
//
// The [Repository] can load assets from any read-only [fs.FS] (e.g. an [embed.FS]) or from raw []bytes.
//
// It resolves dependencies automatically and supports templates overriding.
//
// # Concurrency
//
// Once loaded, a [Repository] is safe for concurrent use by [Repository.Get].
//
// # Dependencies
//
// This package is exposed as an independent module and imposes few dependencies.
//
// The core functionality is built on top of the standard library [text/template].
//
// Excluding test dependencies, it relies directly on the following github.com/fredbi/core packages:
//
//   - github.com/fredbi/core/swag/fs
//   - github.com/fredbi/core/swag/mangling
package repo
