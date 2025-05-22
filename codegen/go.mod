module github.com/fredbi/core/codegen

replace (
	// github.com/fredbi/core/json => ../json
	// github.com/fredbi/core/jsonschema => ../jsonschema
	github.com/fredbi/core/swag/conv => ../swag/conv
	github.com/fredbi/core/swag/mangling => ../swag/mangling
	github.com/fredbi/core/swag/pools => ../swag/pools
	github.com/fredbi/core/swag/typeutils => ../swag/typeutils
// github.com/fredbi/core/templates-repo => ../templates-repo
)

go 1.24.2
