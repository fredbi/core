module github.com/fredbi/core/json/benchmarks

go 1.25

require (
	github.com/fredbi/core/json v0.0.0-00010101000000-000000000000
	github.com/go-json-experiment/json v0.0.0-20250910080747-cc2cfa0554c3
	github.com/mailru/easyjson v0.9.2
)

require (
	github.com/fredbi/core/swag/conv v0.0.0-00010101000000-000000000000 // indirect
	github.com/fredbi/core/swag/pools v0.0.0-00010101000000-000000000000 // indirect
	github.com/josharian/intern v1.0.0 // indirect
)

replace (
	github.com/fredbi/core/json => ../
	github.com/fredbi/core/swag => ../../swag
	github.com/fredbi/core/swag/conv => ../../swag/conv
	github.com/fredbi/core/swag/pools => ../../swag/pools
	github.com/fredbi/core/swag/typeutils => ../../swag/typeutils
)
