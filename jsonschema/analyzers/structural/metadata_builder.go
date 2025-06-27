package structural

import "github.com/fredbi/core/json/stores"

type MetadataBuilder struct {
	s   stores.Store
	err error
	m   Metadata
}

func MakeMetadataBuilder() MetadataBuilder {
	return MetadataBuilder{}
}

func (b MetadataBuilder) WithStore(s stores.Store) MetadataBuilder {
	b.s = s

	return b
}

func (b MetadataBuilder) From(analyzed AnalyzedSchema) MetadataBuilder {
	b.s = analyzed.Metadata().Store()
	b.m = analyzed.Metadata()

	return b
}

func (b MetadataBuilder) WithTitle(title string) MetadataBuilder {
	b.m.title = b.s.PutValue(stores.MakeStringValue(title))

	return b
}

func (b MetadataBuilder) WithDescription(description string) MetadataBuilder {
	b.m.title = b.s.PutValue(stores.MakeStringValue(description))

	return b
}

func (b MetadataBuilder) Metadata() Metadata {
	if b.err == nil {
		return b.m
	}

	return Metadata{}
}
