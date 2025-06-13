package jsonschema

import (
	"io"
	"iter"
	"slices"

	"github.com/fredbi/core/json"
)

// Overlay is an overlay specification for a JSON schema.
type Overlay struct {
	json.Document
}

func MakeOverlay(...Option) Overlay {
	return Overlay{} // TODO
}

// ApplyTo a [Schema] a JSON schema overlay.
func (o Overlay) ApplyTo(_ Schema) Schema {
	return Schema{} // TODO
}

type OverlayCollection struct {
	options
	overlays []Overlay
}

func MakeOverlayCollection(cap int, _ ...Option) OverlayCollection {
	return OverlayCollection{} // TODO
}

func (c OverlayCollection) Len() int {
	return len(c.overlays)
}

func (c OverlayCollection) Overlays() iter.Seq[Overlay] {
	return slices.Values(c.overlays)
}

func (c OverlayCollection) Overlay(index int) Overlay {
	return c.overlays[index]
}

func (c OverlayCollection) DecodeAppend(reader io.Reader) error {
	overlay := MakeOverlay(withOptions(c.options))
	if err := overlay.Decode(reader); err != nil {
		return err
	}
	c.overlays = append(c.overlays, overlay)

	return nil
}

// ApplyTo applies a collection of overlays to a collection of schemas.
func (o OverlayCollection) ApplyTo(_ Collection) Collection {
	return Collection{}
}
