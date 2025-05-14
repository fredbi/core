// Package lexer provides a JSON lexer.
//
// The lexer splits a JSON input stream or a slice of bytes into tokens [token.T] (or [token.VT] for verbatim support).
//
// It checks that the input JSON is grammatically correct (so technically, this is a "parser").
//
// It keeps the context of errors.
//
// The lexer provides a low-level interface for projects which want to manipulate JSON directly,
// and do not necessarily want to unmarshal into go data structures.
//
// The lexer is designed to be low on memory usage: it should never need to allocate more memory
// than your longest string or number value in a stream.
package lexer
