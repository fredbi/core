module github.com/fredbi/core/json/benchmarks

go 1.24.2

require github.com/fredbi/core/json v0.0.0-00010101000000-000000000000

require (
	github.com/fredbi/core/swag/conv v0.0.0-00010101000000-000000000000 // indirect
	github.com/fredbi/core/swag/pools v0.0.0-00010101000000-000000000000 // indirect
)

replace (
	github.com/fredbi/core/json => ../
	github.com/fredbi/core/swag => ../../swag
	github.com/fredbi/core/swag/conv => ../../swag/conv
	github.com/fredbi/core/swag/pools => ../../swag/pools
	github.com/fredbi/core/swag/typeutils => ../../swag/typeutils
)
