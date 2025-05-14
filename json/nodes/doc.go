// Package nodes declares node types for JSON documents.
//
// There are four types of nodes:
//
//   - objects
//   - arrays
//   - scalars (i.e. string, bool or number)
//   - null
//
// Notice that "null" is considered as a type and not a value: a node of type null may only
// be represented by the token "null".
//
// The package light provides an implementation to support nodes used by json documents.
package nodes
