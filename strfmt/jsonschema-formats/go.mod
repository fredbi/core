module github.com/fredbi/core/strfmt/standard-formats

go 1.24.2

require (
	github.com/fredbi/core/strfmt v0.0.0
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.10.0
	go.mongodb.org/mongo-driver v1.17.3
	golang.org/x/net v0.40.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/fredbi/core/strfmt => ../
