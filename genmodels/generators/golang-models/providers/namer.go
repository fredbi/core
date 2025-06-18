package providers

/*
import "github.com/fredbi/core/jsonschema/analyzers/structural"

// TODO: this is a common feature => move to codegen tools

type NameDeconflicter struct {
	namespace map[string]struct{}
}

func NewNameDeconflicter() *NameDeconflicter {
	return &NameDeconflicter{
		namespace: make(map[string]struct{}),
	}
}

func (d *NameDeconflicter) RegisterName(name string) {
	d.namespace[name] = struct{}{}
}

func (d *NameDeconflicter) NameAlreadyExists(name string) bool {
	_, ok := d.namespace[name]

	return ok
}

func (d *NameDeconflicter) Deconflict(name string, namedSchema structural.AnalyzedSchema) (string, bool) {
	uniqueName := name
	assertNameSchema(namedSchema)

	for attempts := 1; ; attempts++ {
		assertNotInfiniteAttempts(attempts)
		if _, isDuplicate := d.namespace[uniqueName]; isDuplicate {
			uniqueName = d.tryDeconflict(uniqueName, namedSchema, attempts)

			continue
		}

		return uniqueName, uniqueName != name
	}
}

func (d *NameDeconflicter) tryDeconflict(name string, _ structural.AnalyzedSchema, attempts int) string {
	// use schema as a context to find our best move
	return name + "X" // TODO
}
*/
