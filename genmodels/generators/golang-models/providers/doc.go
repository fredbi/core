// Package providers equips a models generator with specialized sidekicks.
//
// [NameProvider] specializes in naming things for golang (packages, files, types, consts or vars).
//
// The [NameProvider] methods are injected as callbacks into the [structural.Analyzer] and the schema Builder.
//
// When the [structural.Analyis] runs the bundling, every node (package or schema) is visited and the callback
// invoked.
package providers
