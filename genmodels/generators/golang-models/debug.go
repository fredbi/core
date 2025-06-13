package models

import (
	"errors"
	"fmt"
	"io"
)

// dumpTemplates writes genmodel templates structure to the standard output.
//
// This is useful to produce documentation.
func (g *Generator) dumpTemplates() error {
	return g.generator.Templates().Dump(g.dumpOutput)
}

// dumpAnalyzed dumps the bundled & analyzed input schemas as JSON.
//
// This may be used for debugging.
func (g *Generator) dumpAnalyzed() error {
	if dumper, ok := g.analyzer.(interface{ Dump(io.Writer) error }); ok {
		err := dumper.Dump(g.dumpOutput)

		return errors.Join(err, ErrModel)
	}

	return fmt.Errorf("analyzer %T does not support spec dump: %w", g.analyzer, ErrModel)
}
