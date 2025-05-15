package json

import "slices"

// DocumentCollection is a collection of [Document] s, that share the same options.
//
// It marshals as an array of [Document]s (TODO).
type DocumentCollection struct {
	options

	documents []document
}

func NewDocumentCollection(opts ...Option) *DocumentCollection {
	return &DocumentCollection{
		options: optionsWithDefaults(opts),
	}
}

func (c *DocumentCollection) Append(d Document) {
	c.documents = append(c.documents, d.document)
}

func (c *DocumentCollection) Concat(d DocumentCollection) {
	c.documents = slices.Concat(c.documents, d.documents)
}
