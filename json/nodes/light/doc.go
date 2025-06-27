// Package light exposes an implementation for nodes.
//
// Nodes constitute the building block for JSON documents.
//
// Nodes are compact: they keep only minimal context information,
// so the whole hierarchy requires from the ultimate parent (e.g. a JSON document)
// to hold this context for all nodes.
package light
