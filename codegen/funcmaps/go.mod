module github.com/fredbi/core/codegen/funcmaps

go 1.24.2

replace (
	// github.com/fredbi/core/json => ../json
	// github.com/fredbi/core/jsonschema => ../jsonschema
	// github.com/fredbi/core/swag/conv => ../../swag/conv
	github.com/fredbi/core/swag/mangling => ../../swag/mangling
// github.com/fredbi/core/swag/pools => ../swag/pools
// github.com/fredbi/core/swag/typeutils => ../swag/typeutils
// github.com/fredbi/core/templates-repo => ../templates-repo
)

require (
	github.com/fredbi/core/swag/mangling v0.0.0-00010101000000-000000000000
	github.com/fredbi/core/swag/stringutils v0.0.0-20250515102727-3befc1aaa0d7
	github.com/fredbi/core/templates-repo v0.0.0-20250515102727-3befc1aaa0d7
	github.com/kr/pretty v0.3.1
)

require (
	github.com/kr/text v0.2.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
)
