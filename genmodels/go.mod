module github.com/fredbi/core/genmodels

go 1.24.2

require (
	github.com/fredbi/core/codegen/genapp v0.0.0-20250515102727-3befc1aaa0d7
	github.com/fredbi/core/codegen/templates-repo v0.0.0-20250515102727-3befc1aaa0d7
	github.com/fredbi/core/json v0.0.0-00010101000000-000000000000
	github.com/fredbi/core/jsonschema v0.0.0-00010101000000-000000000000
	github.com/fredbi/core/mangling v0.0.0-20250515102727-3befc1aaa0d7
	github.com/fredbi/core/swag/loading v0.0.0-00010101000000-000000000000
	github.com/spf13/afero v1.14.0
	github.com/stretchr/testify v1.10.0
	golang.org/x/sync v0.15.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fredbi/core/codegen/funcmaps v0.0.0-00010101000000-000000000000 // indirect
	github.com/fredbi/core/swag/conv v0.0.0-00010101000000-000000000000 // indirect
	github.com/fredbi/core/swag/fs v0.0.0-20250515102727-3befc1aaa0d7 // indirect
	github.com/fredbi/core/swag/pools v0.0.0-00010101000000-000000000000 // indirect
	github.com/fredbi/core/swag/stringutils v0.0.0-20250515102727-3befc1aaa0d7 // indirect
	github.com/fredbi/core/swag/typeutils v0.0.0-20250515102727-3befc1aaa0d7 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/mod v0.25.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	golang.org/x/tools v0.34.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/fredbi/core/codegen/funcmaps => ../codegen/funcmaps
	github.com/fredbi/core/codegen/genapp => ../codegen/genapp
	github.com/fredbi/core/codegen/templates-repo => ../codegen/templates-repo
	github.com/fredbi/core/json => ../json
	github.com/fredbi/core/jsonschema => ../jsonschema
	github.com/fredbi/core/mangling => ../mangling
	github.com/fredbi/core/swag/conv => ../swag/conv
	github.com/fredbi/core/swag/fs => ../swag/fs
	github.com/fredbi/core/swag/loading => ../swag/loading
	github.com/fredbi/core/swag/pools => ../swag/pools
	github.com/fredbi/core/swag/stringutils => ../swag/stringutils
	github.com/fredbi/core/swag/typeutils => ../swag/typeutils
)
