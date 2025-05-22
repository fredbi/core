module github.com/fredbi/core/codegen/genapp

go 1.24.2

replace (
	github.com/fredbi/core/codegen/funcmaps => ../funcmaps
	github.com/fredbi/core/codegen/templates-repo => ../templates-repo
	github.com/fredbi/core/swag/fs => ../../swag/fs
	github.com/fredbi/core/swag/mangling => ../../swag/mangling
	github.com/fredbi/core/swag/typeutils => ../swag/typeutils
)

require (
	github.com/fredbi/core/codegen/funcmaps v0.0.0-00010101000000-000000000000
	github.com/fredbi/core/codegen/templates-repo v0.0.0-00010101000000-000000000000
	github.com/spf13/afero v1.14.0
	github.com/stretchr/testify v1.10.0
	golang.org/x/tools v0.33.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fredbi/core/swag/fs v0.0.0-20250515102727-3befc1aaa0d7 // indirect
	github.com/fredbi/core/swag/mangling v0.0.0-00010101000000-000000000000 // indirect
	github.com/fredbi/core/swag/stringutils v0.0.0-20250515102727-3befc1aaa0d7 // indirect
	github.com/fredbi/core/templates-repo v0.0.0-20250515102727-3befc1aaa0d7 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
