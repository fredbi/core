package json

// DocumentCollection is a collection of [Document] s.
//
// It marshals as an array of [Document]s.
type DocumentCollection struct {
	options

	documents []document
}

func NewDocumentCollection(opts ...Option) *DocumentCollection {
	return &DocumentCollection{
		options: optionsWithDefaults(opts),
	}
}

func (c *DocumentCollection) Add(d Document) {
	c.documents = append(c.documents, d.document)
}
