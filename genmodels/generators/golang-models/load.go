package models

import (
	"errors"
	"fmt"

	"github.com/fredbi/core/jsonschema"
	"github.com/fredbi/core/swag/loading"
)

// loadJSONSchemas loads JSON schema definitions from file or URL.
func (g *Generator) loadJSONSchemas() error {
	if g.inputSchemas.Len() > 0 {
		return nil // already provided
	}

	if len(g.sourceSchemas) == 0 {
		return fmt.Errorf("no input schema provided: %w", ErrInit)
	}

	for _, source := range g.sourceSchemas {
		reader := loading.ReaderFromFileOrHTTP(source)
		if err := g.inputSchemas.DecodeAppend(reader); err != nil { // validate input schemas
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

	for _, source := range g.sourceOverlays {
		reader := loading.ReaderFromFileOrHTTP(source)
		if err := g.overlaySchemas.DecodeAppend(reader); err != nil { // validate overlay schema
			return errors.Join(err, ErrModel)
		}

	}

	// apply overlays
	overriddenSchemas := jsonschema.MakeCollection(g.inputSchemas.Len(), jsonschema.WithStore(g.store))

	for i, schema := range g.inputSchemas.Schemas() {
		if i >= g.overlaySchemas.Len() {
			break
		}

		overlay := g.overlaySchemas.Overlay(i)
		overriddenSchemas.Append(overlay.ApplyTo(schema))
		// TODO: add some kind of audit trail in the schema.
	}

	g.inputSchemas = overriddenSchemas

	return nil
}
