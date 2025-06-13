package models

import (
	"errors"
	"fmt"

	"github.com/fredbi/core/swag/loading"
)

// loadJSONSchemas loads JSON schema definitions from file or URL.
func (g *Generator) loadJSONSchemas() error {
	if g.inputSchemas.Len() > 0 {
		return nil // already provided by options
	}

	if len(g.sourceSchemas) == 0 {
		return fmt.Errorf("no input schema provided: %w", ErrInit)
	}

	// load schema resources from local files or over http(s). JSON and YAML files are supported.
	for _, source := range g.sourceSchemas {
		reader := loading.ReaderFromFileOrHTTP(source, g.loadOptions...)
		if err := g.inputSchemas.DecodeAppend(reader); err != nil { // decode and validate input schemas
			return errors.Join(err, ErrModel)
		}
	}

	return nil
}

// loadOverlays loads overlays from file or URL, and applies overlays to schemas.
func (g *Generator) loadOverlays() error {
	if g.overlaySchemas.Len() > 0 {
		return nil // already provided
	}

	// load overlay resources from local files or over http(s). JSON and YAML files are supported.
	for _, source := range g.sourceOverlays {
		reader := loading.ReaderFromFileOrHTTP(source, g.loadOptions...)
		if err := g.overlaySchemas.DecodeAppend(reader); err != nil { // decode and validate overlay schemas
			return errors.Join(err, ErrModel)
		}
	}

	// apply overlays in bulk
	//
	// NOTE: [jsonschema.OverlayCollection.ApplyTo] does not return errors, as any validation error
	// in this document has already been captured during "DecodeAppend".
	//
	// JSON path expressions are valid. The ones that do not resolve to any part of a schema are ignored.
	g.inputSchemas = g.overlaySchemas.ApplyTo(g.inputSchemas)

	return nil
}

/*
Status:

* loading.ReaderFromFileOrHTTP(source) : TODO in swag/loading
* jsonschema.Collection.DecodeAppend(reader) : TODO in jsonschema
* jsonschema.OverlayCollection.ApplyTo(jsonschema.Collection) : TODO in jsonschema

*/
