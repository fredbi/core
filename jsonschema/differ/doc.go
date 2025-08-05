// Package differ knowns how to Diff and Patch a pair of [jsonschema.Schema] s.
//
// # Diff
//
// Differences are qualified to detect breaking changes or backward-compatible changes.
//
// # Patch
//
// A [jsonschema.Overlay] is produced that when applied to the first [jsonschema.Schema], yields the second one.
package differ
