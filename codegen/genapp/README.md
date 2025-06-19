# genapp [![Go Reference](https://pkg.go.dev/badge/github.com//fredbi/core/codegen/genapp.svg)](https://pkg.go.dev/github.com/fredbi/core/codegen/genapp)
[![codecov](https://codecov.io/github/fredbi/core/branch/master/graph/badge.svg?flag=codegen-genapp)](https://codecov.io/github/fredbi/core?flag=codegen-genapp)

A general-purpose golang code generator.

### Features

* embeds a [templates repository](../templates-repo/README.md)
* formats and check imports using `golang.org/x/tools/import`
* resolve package path for a target folder, even outside the go build tree, using `golang.org/x/tools/go/packages`
* expose a single `Render` method to execute a template and format the resulting go code

Several options are available to customize this app builder.

### See it in action

* [JSON schema models generator](../../genmodels/generators/golang-models/README.md)
* TODO: [testing generated code](../gentesting/README.md)

### Background and credits

This package has been built after the internal tooling built for [go-swagger](https://goswagger.io).

The motivation for this package was to factorize out that valuable component, so as to be able to build other powerful code generation
tools independently from go-swagger. Further improvements may be added independently.

The original version may be found [here](https://github.com/go-swagger/go-swagger/tree/master/generator/support.go#L129).

There are a few differences though:

* this version is intended to be very generic and has been stripped from any target-dependent settings
* all it knows is that is that it should use a templates Repository to render (primarily) some go code
* rendering may be called concurrently
* it maintains it own set of optional settings
* it provides a helper to generate go modules with their own go.mod
* it can build go module-aware apps, anywhere on the building system
* the package path resolution originally made [here](https://github.com/go-swagger/go-swagger/blob/5226f2c6fcc7705caaab26862c941370699dbd97/generator/language.go#L305)
  is complex and difficult to keep in line with go build evolutions. Our implementation hands over the heavy lifting
  to `golang.org/x/tools/go/packages` 

Notice that much of the go specifics formerly found in https://github.com/go-swagger/go-swagger/blob/master/generator/language.go
have now moved to [the new name mangler](../../mangling/README.md).
