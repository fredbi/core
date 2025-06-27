package structural

import (
	"iter"

	"github.com/fredbi/core/jsonschema/analyzers"
	"github.com/fredbi/core/jsonschema/analyzers/internal/graph/v2"
)

type packageTree struct {
	graph.Tree[analyzers.UniqueID, AnalyzedPackage, struct{}]
}

func newPackageTree() *packageTree {
	return &packageTree{}
}

func (p *packageTree) AddPath(pth string) {
}

func (p *packageTree) PackageByID(id analyzers.UniqueID) (AnalyzedPackage, bool) {
	return AnalyzedPackage{}, false
}

func (p *packageTree) Leaves() iter.Seq[AnalyzedPackage] {
	return nil
}

func (p *packageTree) TraverseDFS() iter.Seq[AnalyzedPackage] {
	return nil
}

func (p *packageTree) TraverseBFS() iter.Seq[AnalyzedPackage] { // NO: this is for DAG
	return nil
}

func (p *packageTree) Inverted() *packageTree { // TODO: does not return a tree but a DAG
	return nil
}

type schemaGraph struct {
	graph.DAG[analyzers.UniqueID, AnalyzedSchema, AnalyzedSchemaContext]
}

func newSchemaGraph() *schemaGraph {
	return &schemaGraph{}
}

func (s *schemaGraph) SchemaByID(id analyzers.UniqueID) (AnalyzedSchema, bool) {
	return AnalyzedSchema{}, false
}

func (s *schemaGraph) SchemaByPath(pth string) (AnalyzedSchema, bool) {
	return AnalyzedSchema{}, false
}
