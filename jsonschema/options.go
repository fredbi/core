package jsonschema

type Option func(*options)

type options struct {
	version Version
}

func WithVersion(version Version) Option {
	return func(o *options) {
		o.version = version
	}
}

/*
func WithVersion(version SchemaVersion) Option {
	return func(o *options) {
		o.version = version
	}
}
*/
