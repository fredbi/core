module github.com/fredbi/core/templates-repo/funcmaps/golang

go 1.24.2

require (
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/go-openapi/inflect v0.21.2
	github.com/go-openapi/swag v0.23.1
	github.com/kr/pretty v0.3.1
	github.com/stretchr/testify v1.10.0
	golang.org/x/tools v0.1.12
)

require (
	dario.cat/mergo v1.0.1 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/imdario/mergo v1.0.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/sys v0.23.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/fredbi/core/swag/mangling => ../../../swag/mangling
	github.com/imdario/mergo v1.0.2 => dario.cat/mergo v1.0.2
)
