package differ

import "github.com/fredbi/core/jsonschema"

type Differ struct {
}

func New(opts ...Option) *Differ {
	return &Differ{}
}

func (d *Differ) Diff(old, new jsonschema.Schema) Result {
	return Result{} // TODO
}
