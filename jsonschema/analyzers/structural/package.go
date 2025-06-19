package structural

// AnalyzedPackage is the outcome of a package when bundling a JSON schema.
//
// Package hierarchy is a tree, not just a DAG.
type AnalyzedPackage struct {
	analyzedObject

	schemas        []*AnalyzedSchema // schemas defined in this package
	parent         *AnalyzedPackage
	children       []*AnalyzedPackage
	ultimateParent *AnalyzedPackage
	extensions     Extensions
}

func (p AnalyzedPackage) IsEmpty() bool {
	return p.ID() == ""
}

func (p AnalyzedPackage) Parent() AnalyzedPackage {
	if p.parent != nil {
		return *p.parent
	}

	return AnalyzedPackage{}
}

func (p AnalyzedPackage) Children() []AnalyzedPackage {
	values := make([]AnalyzedPackage, len(p.children))
	for i, child := range p.children {
		values[i] = *child
	}

	return values
}

func (p AnalyzedPackage) Schemas() []AnalyzedSchema {
	values := make([]AnalyzedSchema, len(p.schemas))
	for i, schema := range p.schemas {
		values[i] = *schema
	}

	return values
}

func (p AnalyzedPackage) GetExtension(extension string, aliases ...string) (any, bool) {
	return p.extensions.Get(extension, aliases...)
}
