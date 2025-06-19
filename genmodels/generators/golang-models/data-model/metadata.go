package model

import (
	types "github.com/fredbi/core/genmodels/generators/extra-types"
	"github.com/fredbi/core/jsonschema/analyzers"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

// Metadata collected about a schema.
type Metadata struct {
	structural.Metadata

	Related     []analyzers.UniqueID      // related types (e.g container -> children). Used in doc string.
	Report      []types.InformationReport // generation report: warnings, tracking decisions etc
	Annotations []string                  // reverse-spec annotations, e.g. "swagger:model"
}
