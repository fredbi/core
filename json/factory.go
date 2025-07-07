package json

// DocumentFactory is a factory that produces [Document] s with the same settings.
//
// TODO: the idea with a factory is to be able to keep track of all things generated so we may recycle them
type DocumentFactory struct {
	options
}

// NewDocumentFactory builds a factory for [Document] s
func NewDocumentFactory(opts ...Option) *DocumentFactory {
	return &DocumentFactory{
		options: optionsWithDefaults(opts),
	}
}

func (f DocumentFactory) Empty() Document {
	return Document{
		options: f.options,
	}
}

func (f DocumentFactory) Clone(d Document) Document {
	clone := Document{
		d.options,
		d.document,
	}
	clone.store = d.store

	return clone
}
