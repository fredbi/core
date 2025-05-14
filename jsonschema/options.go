package jsonschema

type Option func(*options)

type options struct {
	version SchemaVersion
}

func WithVersion(version SchemaVersion) Option {
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
