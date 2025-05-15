module github.com/fredbi/core/templates-repo/funcmaps/golang

go 1.24.2

require (
	github.com/fredbi/core/swag/mangling v0.23.1
	github.com/fredbi/core/swag/stringutils v0.23.1
	github.com/fredbi/core/templates-repo v0.23.1
	github.com/kr/pretty v0.3.1
)

require (
	github.com/kr/text v0.2.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
)

replace (
	github.com/fredbi/core/swag/mangling => ../../../swag/mangling
	github.com/fredbi/core/swag/stringutils => ../../../swag/stringutils
	github.com/fredbi/core/templates-repo => ../..
	github.com/imdario/mergo v1.0.2 => dario.cat/mergo v1.0.2
)
