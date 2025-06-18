module github.com/fredbi/core/codegen/funcmaps

go 1.24.2

require (
	github.com/fredbi/core/mangling v0.0.0-00010101000000-000000000000
	github.com/fredbi/core/swag/stringutils v0.0.0-20250515102727-3befc1aaa0d7
	github.com/fredbi/core/swag/typeutils v0.0.0-20250515102727-3befc1aaa0d7
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/fredbi/core/mangling => ../../mangling
	github.com/fredbi/core/swag/stringutils => ../../swag/stringutils
	github.com/fredbi/core/swag/typeutils => ../../swag/typeutils
)
