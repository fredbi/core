package genapp

// ModOption configures settings for the generated go module.
type ModOption func(*modOptions)

type modOptions struct {
	modulePath string
	goVersion  string
}

func modOptionsWithDefaults(opts []ModOption) modOptions {
	var o modOptions

	for _, apply := range opts {
		apply(&o)
	}

	return o
}

// WithModulePath overrides the default go module fully qualified name when creating a go mod.
//
// This is equivalent to the command "go mod init {pth}".
func WithModulePath(pth string) ModOption {
	return func(o *modOptions) {
		o.modulePath = pth
	}
}

// WithGoVersion overrides the default go module requirement on the required minimum version of the go compiler.
//
// This is equivalent to the command "go mod tidy -go={version}".
func WithGoVersion(version string) ModOption {
	return func(o *modOptions) {
		o.goVersion = version
	}
}
