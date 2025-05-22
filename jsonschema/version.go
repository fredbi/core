package jsonschema

type Version uint8

const (
	VersionUndefined Version = iota
	VersionDraft4
	VersionDraft5
	VersionDraft6
	VersionDraft7
	VersionDraft2019
	VersionDraft2020
	VersionOpenAPIv2
	VersionOpenAPIv300
	VersionOpenAPIv301
	VersionOpenAPIv302
	VersionOpenAPIv303
	VersionOpenAPIv304
	VersionOpenAPIv310
	VersionOpenAPIv4draft
)

func (v Version) String() string {
	switch v {
	case VersionUndefined:
		return "undefined"
	default:
		// TODO
		panic("yay")
	}
}

func (v Version) MetaSchemaURL() string {
	return ""
}

func (v Version) DocURL() string {
	return ""
}
