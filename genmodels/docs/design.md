# genmodels design

## Non-goals

* the accurate **understanding of JSON schema** is handed over to [our JSON schema library](../jsonschema/README.md)
* producing **documentation artifacts** (not godoc) is handed over to [our doc generator](../gendoc/README.md)
* the **reverse-engineering of go types** into JSON schema is handed over to [our spec generator](../genspec/README.md)
* **faking JSON schema data** is handed over to [our stubs generator](../stubs/README.md)

## Roles and responsibilities

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
