package faker

import "github.com/fredbi/core/spec"

// Faker generates a random OpenAPI spec.
type Faker struct {
}

func New(opts ...Option) Faker {
	return Faker{}
}

type Generated struct {
	spec.Spec
	valid bool
}

func (g Generated) ShouldBeValid() bool {
	return g.valid
}

func (f Faker) Generate() Generated {
	return Generated{Spec: spec.Make()} // TODO
}

func (f Faker) GenerateMany(n int) []Generated {
	return []Generated{} // TODO
}
