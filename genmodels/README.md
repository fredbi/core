# genmodels

A go module and a minimal CLI to build code artifacts from JSON schemas.

## Goals

This package is primarily intended to generate go types ("models") from JSON schemas,
although it may be extended to produce other artifacts.

The goal of this package is to automatically produce idiomatic go code that verifies the following, in that order:

1. correctly implements the JSON schema specification
2. remains clear and readable
3. is fast to serialize, deserialize and validate
4. passes `go vet` and the `revive` linter
4. is well documented with godoc
5. abides by most sensible linting rules (e.g. provided by `golangci-lint`)

## Quick start

#### Installation

To install the `genmodels` [CLI](#CLI-usage) from source:

```cmd
go install github.com/fredbi/core/genmodels/cmd/...@latest
```

#### Example usage

## JSON schema dialects

It supports the following dialects of JSON schema:

* jsonschema.org specifications:

  * draft 4 (aka draft 5)
  * draft 6
  * draft 7
  * draft 2019
  * draft 2020

* dialects defined by the OpenAPI specifications:

  * OpenAPI v2 (aka Swagger) schema and simple schema (parameters and response headers)
  * OpenAPI v3.0 schema and simple schema
  * OpenAPI v3.1 schema and simple schema

For more about the specifics of the above dialects, refer to package [`jsonschema`](../jsonschema/README.md),
and specifically how that package [analyzes the structure of a JSON schema](../jsonschema/analyzers/structural/README.md).

## [CLI usage](./docs/cli.md)

## [Features](docs/features.md)

## [How does it work?](docs/design.md)

## [Dependencies](docs/dependencies.md)

## [About `null`, `nil` and the use of pointers](docs/null.md)

## Contributions

TODO(fredbi): explain how contrib should work
