package jsonschema

import (
	"github.com/fredbi/core/json/stores"
)

type Option func(*options)

type options struct {
	version Version
	store   stores.Store
}

func WithVersion(version Version) Option {
	return func(o *options) {
		o.version = version
	}
}

func WithStore(s stores.Store) Option {
	return func(o *options) {
		o.store = s
	}
}

func withOptions(opts options) Option {
	return func(o *options) {
		*o = opts
	}
}

/*
func WithVersion(version SchemaVersion) Option {
	return func(o *options) {
		o.version = version
	}
}
*/
