module github.com/fredbi/core/codegen/funcmaps

go 1.24.2

require (
	github.com/fredbi/core/swag/mangling v0.0.0-00010101000000-000000000000
	github.com/fredbi/core/swag/stringutils v0.0.0-20250515102727-3befc1aaa0d7
	github.com/fredbi/core/swag/typeutils v0.0.0-20250515102727-3befc1aaa0d7
	github.com/kr/pretty v0.3.1
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/fredbi/core/swag/mangling => ../../swag/mangling
	github.com/fredbi/core/swag/stringutils => ../../swag/stringutils
	github.com/fredbi/core/swag/typeutils => ../../swag/typeutils
)
