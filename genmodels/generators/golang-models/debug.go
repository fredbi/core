package models

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// dumpTemplates writes genmodel templates structure to the standard output.
//
// This is useful to produce documentation.
func (g *Generator) dumpTemplates() error {
	return g.inner.Templates().Dump(os.Stdout)
}

// dumpAnalyzed dumps the bundled & analyzed input schemas as JSON.
//
// This may be used for debugging.
func (g *Generator) dumpAnalyzed() error {
	if dumper, ok := g.analyzer.(interface{ Dump(io.Writer) error }); ok {
		err := dumper.Dump(os.Stdout)

		return errors.Join(err, ErrModel)
	}

	return fmt.Errorf("analyzer %T does not support spec dump: %w", g.analyzer, ErrModel)
}
