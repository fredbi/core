package models

// loadConfig loads and merge the codegen config from file, flags or options passed to [New] (in this order).
//
// TODO
func (g *Generator) loadConfig() error {
	// load config file

	// apply CLI flags

	// apply options in new
	// TODO: check fs vs g.TargetImportPath
	// options overrides
	if g.outputPath != "" {
		g.TargetImportPath = g.outputPath
	}

	// TODO fs

	return g.validateOptions()
}
