module github.com/fredbi/core/json

go 1.24.2

replace (
	github.com/fredbi/core/swag => ../swag
	github.com/fredbi/core/swag/conv => ../swag/conv
	github.com/fredbi/core/swag/pools => ../swag/pools
	github.com/fredbi/core/swag/typeutils => ../swag/typeutils
)

require (
	github.com/fredbi/core/swag/conv v0.0.0-00010101000000-000000000000
	github.com/fredbi/core/swag/pools v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.10.0
	go.step.sm/crypto v0.62.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
