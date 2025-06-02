# genmodels

A go module and a minimal CLI to build code artifacts from JSON schemas.

## Goals

This package is primarily intended to generate go types ("models") from JSON schemas,
although it may be extended to produce other artifacts.

The goal of this package is to produce idiomatic go that:

1. correctly implements the JSON schema specification
2. remains clear and readable
3. is fast
4. passes `go vet` and the `revive` linter
4. is documented with godoc
5. abides by most sensible linting rules (e.g. provided by `golangci-lint`)

TODO(fredbi): refactor package as:
```
 ./generators/golang-models/templates
 ./generators/protobuf-models
 ./generators/extra-types
 ./generators/common-settings
 ./generators/internal/sync
 ./contrib
```

TODO(fredbi): explain how contrib should work

#### Non-goals

* the accurate **understanding of JSON schema** is handed over to [our JSON schema library](../jsonschema/README.md)
* producing **documentation artifacts** (not godoc) is handed over to [our doc generator](../gendoc/README.md)
* the **reverse-engineering of go types** into JSON schema is handed over to [our spec generator](../genspec/README.md)
* **faking JSON schema data** is handed over to [our stubs generator](../stubs/README.md)

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

## CLI usage

This module provides a standalone CLI tool that can be used to process JSON schemas or OpenAPI schemas.

The `go-swagger` package provides a more integrated set of tools to work with OpenAPI specs, schemas, etc.

The `genmodels` tool is simpler and more limited.

## Features

=> defer to features.md
* supports all JSON schema types, including objects, arrays and tuples
* supports `allOf`, `anyOf`, `oneOf` constructs, including OpenAPI's notion of inheritance (types with a `discriminator`)
* supports extensible types, using `additionalProperties` or `additionalItems` 
* support custom types
* support streaming with `io.Reader` and `io.Writer`
* fully configurable generation options, down to the schema level, using generation options or `overlay schemas` (e.g. with `x-go-*` directives)
* generated marshalers/unmarshalers
* enum types and constants
* custom struct tags
* can generate extra functions and methods, such as constructors, String or DeepCopy methods
* can generate test code alongside your models
* can produce poolable types for better gc performance

TODO: generic types (e.g. with `$dynamicRef`) ?

**Validation**

The generated types know how to validate against their schema.

By default, types know how to `UnmarshalJSONValidate` and `MarshalJSONValidate`, that is
validation occurs only when unmarshaling and marshaling and is carried out in one pass.

Optionally, types may support standalone validation: data may be validated independently from the serialization process.

Another option is to **not** support validation, possibly delegating it to some runtime validator,
like the ones produced by [our validation analyzer](../jsonschema/analyzers/validation/README.md).

**go types that you won't see generated**

=> defer to design.md

* channels
* arrays
* pointers to slices or maps

## How does it work?

#### Roles and responsibilities

**Out of scope**

The models generator focuses on data schemas, not APIs in general.

OpenAPI schemas, including simple schemas for parameters and headers are considered specific dialects of a schema.

The models generator delegates JSON schema specifics to package [`jsonschema`](../jsonschema/README.md).
In particular, it doesn't deserialize or validate JSON schemas.

All its takes is the analyzed input from the [analyzed structure of a schema](../jsonschema/analyzers/structural/README.md), or a collection of schemas.

As a matter of facts, `genmodels` does not _understand_ JSON schema at all, but only the abstract version presented by the analyzer.

If we compare with the corresponding feature in `go-swagger` v0.x, this module covers the "model generation" part.

All the activity carried out there to explore the graph of schemas and subschemas, resolving `$ref`s, and trying to determine
what's coming next in line down this graph is now fully handed over to the analyzer.

This results in much simpler code generation templates, that do not need to be fully recursive.

**In scope**

* Target type mapping

  * `genmodels` selects an appropriate design to render in idiomatic go a given analyzed schema structure.
    * it decides when a schema requires a type definition
    * it owns code generation templates
    * it owns the data structure that hydrates the templates
  * it takes informed decisions about types, selecting a struct, a slice, a map or an interface.
  * it produces type decorators, such as struct tags and godoc-friendly comments

* Naming

  * it produces names that fit with go
  * it decides the package layout
  * it determines and arbitrates name conflicts

* Optional settings

  * it applies all settings that pertain to code generation (options for code layout, choice of marshalers, etc.)

** May be included in scope in the future**

Rendering in the go language is only a particular outcome of the analysis.

Rendering a `protobuf` specification could be another goal.

Rendering for different languages may also be possible, as the analysis carried out upstream remains suitable for
any statically typed language.

#### Dependencies

=> defer to design.md

This tool stitches together features from:

* [`go-openapi/jsonschema`](../jsonschema/README.md)
* [`go-openapi/codegen`](../codegen/README.md)
* [`go-openapi/strfmt`](../strfmt/README.md)
* [`go-openapi/swag`](../swag/README.md)

#### Dependencies of the generated code

=> defer to dependencies.md

All generated dependencies may be overridden by a location of your choice.

* By default:

  * the go standard library
  * other github.com/fredbi/core modules, like:
    * `github.com/fredbi/core/strfmt` (when using `format` for strings)

* Required when enabling some options:

  * `fredbi/core/validate` (validation helpers)
  * `fredbi/core/json/types` (to support the specifics of JSON not natively supported in go, such a `null` or arbitrary large numbers)
  * `fredbi/core/json` (to work with opaque JSON documents or "dynamic JSON" types)
  * YAML marshaling libraries
  * `MarshalBinary` interface
  * `BindFromRequest` and `BindToResponse` methods (for OpenAPI parameters and responses)
  * alternate JSON marshaling libraries (e.g. `mailru/easyjson`, `goccy/go-json`, `go-jsoniterator` or using `../json/lexers`)

## About `null` and the use of pointers

=> defer to design.md

TODO
