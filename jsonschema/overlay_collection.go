package jsonschema

import (
	"io"
	"iter"
	"slices"
)

// OverlayCollection represents a collection of [Overlay] s.
type OverlayCollection struct {
	*overlayOptions
	overlays []Overlay
}

func MakeOverlayCollection(cap int, opts ...OverlayOption) OverlayCollection {
	o := overlayOptionsWithDefaults(opts)
	return OverlayCollection{
		overlayOptions: o,
		overlays:       make([]Overlay, 0, cap),
	}
}

func (c OverlayCollection) Len() int {
	return len(c.overlays)
}

func (c OverlayCollection) Overlays() iter.Seq[Overlay] {
	return slices.Values(c.overlays)
}

func (c *OverlayCollection) Append(overlay Overlay) {
	c.overlays = append(c.overlays, overlay)
}

func (c OverlayCollection) Overlay(index int) Overlay {
	return c.overlays[index]
}

func (c *OverlayCollection) DecodeAppend(reader io.Reader) error {
	overlay := MakeOverlay(withOverlayOptions(c.overlayOptions))
	if err := overlay.Decode(reader); err != nil {
		return err
	}

	c.Append(overlay)

	return nil
}

// ApplyTo applies a collection of overlays to a collection of schemas.
//
// All overlays in the collection are applied in sequence to each schema in the
// passed [Collection] of [Schema] s to form a new [Collection].
//
// Notice that schemas are processed independently.
func (o OverlayCollection) ApplyTo(c Collection) Collection {
	result := CollectionFromTemplate(c)
	for overlay := range o.Overlays() {
		for schema := range c.Schemas() {
			result.Append(overlay.ApplyTo(schema))
		}
	}

	return result
}
