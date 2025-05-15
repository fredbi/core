module github.com/fredbi/core/json/jsonpath

go 1.24.2

replace (
	github.com/fredbi/core/json => ..
	github.com/fredbi/core/swag/conv => ../../swag/conv
	github.com/fredbi/core/swag/pools => ../../swag/pools
	github.com/fredbi/core/swag/typeutils => ../../swag/typeutils
)

require (
	github.com/PaesslerAG/jsonpath v0.1.1
	github.com/fredbi/core/json v0.0.0-00010101000000-000000000000
)

require (
	github.com/PaesslerAG/gval v1.0.0 // indirect
	github.com/fredbi/core/swag/conv v0.0.0-00010101000000-000000000000 // indirect
	github.com/fredbi/core/swag/pools v0.0.0-00010101000000-000000000000 // indirect
	github.com/fredbi/core/swag/typeutils v0.0.0-20250514060830-b48c49a036a1 // indirect
)
