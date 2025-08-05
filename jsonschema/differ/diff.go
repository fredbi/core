package differ

import "github.com/fredbi/core/jsonschema"

// Differ knows how to compare two versions of a json schema.
type Differ struct {
	*options
}

func New(opts ...Option) *Differ {
	return &Differ{}
}

// Diff computes the differences between new and old [jsonschema.Schema] as a diff [Result].
func (d *Differ) Diff(old, new jsonschema.Schema) Result {
	return Result{} // TODO
}

// Diff computes a [jsonschema.Overlay] patch that when applied to old, produces the new schema.
func (d *Differ) Patch(old, new jsonschema.Schema) jsonschema.Overlay {
	return jsonschema.MakeOverlay() // TODO
}
