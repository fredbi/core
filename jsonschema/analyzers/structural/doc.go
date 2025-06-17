// Package structural exposes a structural JSON schema [Analyzer].
//
// The focus of this analysis is the production of serializations, e.g. for strongly typed languages.
//
// This analyzer inspects JSON schema validations to reason about the structure of the schemas:
// namespaces, named vs anonymous schemas and dependencies graph are an outcome of this analysis.
//
// An [Analyzer] knows how to Analyze, which may include some refactoring and to Bundle a [jsonschema.Schema] or [jsonschema.Collection].
//
// # Namespaces
//
// Organized schemas that are constructed uing '$ref', '$id' and '$defs' (or "definitions") imply the notion of namepace
// for the named schemas they refer to.
//
// A namespace represents a set of non-conflicting names.
// The notion of non-conflicting names generalize the notion of
// uniqueness: since the same schema is used to generate different artifacts following different naming conventions
// (e.g. a package name, a generated file name, a type definition...), two different names may still produce conflicts.
//
// The [Analyzer] exposed by this package supports the construction of such namespaces inferred from JSON schema constructs.
//
// The [Analyzer.Bundle] operation allows to reorganize the entire namespace, abtracting away the notion of '$ref'.
//
// # Structural validations
//
// The [Analyzer] is specifically focused on structural validations, that is, JSON schema validations which are useful
// to infer some rules about how to serialize a schema.
//
// Some JSON validations are not very useful in this regard and we consider them as pure validations.
//
// Example: "maxLength" is not useful to extract any information about a type.
//
// Obvious structural validations that we may find in the JSON schema grammar are:
//
//   - 'type'
//   - object 'properties', 'required', 'patternProperties' and 'additionalProperties'
//   - array 'items'
//   - tuple 'prefixItems' and 'items' (formerly 'items' and 'additionalItem' respectively)
//   - composition validations: 'allOf', 'anyOf', 'oneOf', 'not'
//   - 'enum'
//   - 'default'
//   - 'format'
//   - conditional validations such as 'dependentRequired', 'dependentSchema' and 'if'/'then'/'else' constructs
//
// Other validations such as 'minimum', 'minLength', 'propertyNames' etc do not provide information directly, but may help
// determine if an uninitialized type (zero value) is valid for a given schema.
//
// And in addition, since the [Analyzer] supports the OpenAPI dialects of JSON schema:
//   - 'discriminator'
package structural
