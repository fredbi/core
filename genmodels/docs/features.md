# Features

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
