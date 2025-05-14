// Package token defines common JSON token types with their kind.
//
// These data structures should be returned by all implementations of a JSON lexer.
//
// There are two flavors of supported tokens:
//
// - the basic token type [T], that is suitable for most purposes
// - the enriched token type [VT], that keeps track of non-significant white space in the JSON input
//
// The enriched token type should be reserved to specific usage such as rendering verbatim documents
// (e.g. for linters, auto-fixers, editor plugins, etc.)
package token
