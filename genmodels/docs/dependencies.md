# Dependencies

## Dependencies of the genmodels tool

This tool stitches together features from:

* [`go-openapi/jsonschema`](../jsonschema/README.md)
* [`go-openapi/codegen`](../codegen/README.md)
* [`go-openapi/strfmt`](../strfmt/README.md)
* [`go-openapi/swag`](../swag/README.md)

## Dependencies of the generated code

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
