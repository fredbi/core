// Package jsonschema provides tools to work with JSON schemas.
//
// # Schema
//
// The [Schema] type is a particular kind of [json.Document]. All operations supported by [json.Document]
// are available to a [Schema].
// The [Schema] type can deserialize and serialize JSON documents that contain a json schema.
//
// It fully supports:
//
//   - all published JSON schema draft versions, from draft4 up to draft 2020 (latest to this date).
//   - well-known JSON schema dialects defined by the OpenAPI specifications (v2, v3.x)
//
// Optionally, the [Schema] type may support the "$data" extension introduced by the ajv validator.
//
// When deserializing a [Schema] from JSON input, the outcome is necessarily a legit JSON schema or an error,
// meaning that a deserialized [Schema] is fit to work with.
//
// You cannot use directly a [Schema] to validate data. Data validators are built after an analysis by
// an instance of [github.com/fredbi/core/jsonschema/analyzers/validations.Analyzer]. This gives full control over
// how we want to validate JSON data.
// If you just want to use a [Schema] to validate data, this process is wrapped-up with a more convenient API by
// package [github.com/fredbi/core/jsonschema/validator]
//
// # Builder
//
// [Schema] structs are immutable. The schema [Builder] is used to construct schemas programmatically.
// A [Builder] may be used to clone a [Schema] and operates mutations of the original on this clone.
//
// # Overlay
//
// We define an [Overlay] type very similar to OpenAPI's overlay to amend json schema documents.
// An [Overlay] allows developers to inject custom behavior or extensions into existing schemas.
//
// An [Overlay] may also be considered to track version changes.
//
// Package [github.com/fredbi/core/jsonschema/differ] provides a Patch method to compute the [Overlay] to
// apply to one version and obtain the next one.
//
// # JSON references
//
// [Ref] is the type that holds "$ref" and "$dynamicRef" definitions from  a JSON schema.
// A [Ref] knows how to resolve JSON references to local and remote JSON and YAML documents, from a local file
// system or from HTTP(s) URLs.
//
// [Ref] handles cyclical references, JSON schemas "$id", "$anchor" and "$dynamicAnchor".
//
// # Collections
//
// [Schema] s and [Overlay] s can be regrouped in collections, [Collection] and [OverlayCollection] respectively,
// to share settings and perform operations on the whole collection of items.
//
// # YAML support
//
// # TODO
//
// # Other tools
//
//   - Package [github.com/fredbi/core/jsonschema/analyzers/structural] analyzes a [Schema] to build model generators from a JSON schema specification.
//   - Package [github.com/fredbi/core/jsonschema/analyzers/validations] analyzes a [Schema] to build validators or generators.
//   - Package [github.com/fredbi/core/jsonschema/cmd/jsonschema] provides a CLI to use the tools exposed by the jsonschema packages library.
//   - Package [github.com/fredbi/core/jsonschema/converter] provides a Converter type to convert a [Schema] to another json schema version.
//   - Package [github.com/fredbi/core/jsonschema/differ] provides a Differ type to analyze changes between two [Schema] s.
//   - Package [github.com/fredbi/core/jsonschema/faker] provides types to generate fake [Schema] s or fake data that
//   - Package [github.com/fredbi/core/jsonschema/linter] provides a linter for json schemas.
//   - Package [github.com/fredbi/core/jsonschema/validator] wraps all needed analysis to provide a data validator against a [Schema].
//     validate against a given [Schema].
//
// # TODOs
//
//   - [] a Factory to produce garbage-safe [Schema] s that may be recycled from a pool
package jsonschema
