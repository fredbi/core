module github.com/fredbi/core/spec

go 1.23.6

require (
	github.com/fredbi/core/json v0.0.0-00010101000000-000000000000
	github.com/fredbi/core/jsonschema v0.0.0-00010101000000-000000000000
)

require github.com/fredbi/core/swag/pools v0.0.0-00010101000000-000000000000 // indirect

replace (
	github.com/fredbi/core/json => ../json
	github.com/fredbi/core/jsonschema => ../jsonschema
	github.com/fredbi/core/stubs => ../stubs
	github.com/fredbi/core/swag => ../swag
	github.com/fredbi/core/swag/pools => ../swag/pools
)
