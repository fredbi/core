// Package json deals with JSON documents.
//
// This library provides tools to work with raw JSON data, with no attempt to convert it to native go types until
// the last moment.
//
// This allows to abide strictly to the JSON standards. In particular, it supports JSON-specifics which are not
// well handled by the standard library, such as:
//
//   - null
//   - values with a zero value in go (e.g. 0, "", false)
//   - very large or very high-precision numbers that overflow go native types (e.g. int64, float64)
//
// # Main memory usage
//
// The library design is primarily focused on keeping a low memory profile, in terms of space as well as in
// terms of allocations:
//
//   - few things are pointers or require allocations
//   - most data structures that need a dynamic allocation may be recycled using pools, thus amortizing
//     the cost of allocations
//   - JSON values are isolated in a specific in-memory store, which takes care of optimizing their size
//   - object keys are interned
//   - lazy JSON value resolution
//
// # Immutability
//
// Another design goal of this package is immutability: all provided objects [Document], [light.Node], [token.Token],
// [stores.Value] etc are all immutable, and designed to be cheap to clone instead.
//
// Mutating or constructing a JSON [Document] programmatically requires a [Builder] to carry out a series of fluent
// building methods and produce a modified clone of the original [Document].
//
// # Extensibility
//
// There are tons of use-cases out there to play with.
//
// Specific modules may be added to extend or improve the default implementations of [lexers.Lexer], [writers.Writer]
// and [stores.Store]. As independent modules, they may bring with them their own set of dependencies without
// altering the dependencies of the parent module.
package json
