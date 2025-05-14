package faker

import "github.com/fredbi/core/jsonschema"

// Faker generates a random JSONSchema.
type Faker struct {
}

func New(opts ...Option) Faker {
	return Faker{}
}

func (f Faker) Generate() Generated {
	return Generated{Schema: jsonschema.Make()} // TODO
}

type Generated struct {
	jsonschema.Schema
	valid bool
}

func (g Generated) ShouldBeValid() bool {
	return g.valid
}

func (f Faker) GenerateMany(n int) []Generated {
	return []Generated{} // TODO
}
