module github.com/fredbi/core/json/documents/jsonpath

go 1.24.2

replace (
	github.com/fredbi/core/json => ../..
	github.com/fredbi/core/swag/pools => ../../../swag/pools
)

require (
	github.com/PaesslerAG/jsonpath v0.1.1
	github.com/fredbi/core/json v0.0.0-00010101000000-000000000000
)

require (
	github.com/PaesslerAG/gval v1.0.0 // indirect
	github.com/fredbi/core/swag/pools v0.0.0-00010101000000-000000000000 // indirect
)
