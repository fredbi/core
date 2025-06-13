# genapp

A general-purpose golang code generator.

### Credits

This package has been built from the internal tooling built for go-swagger.

The motivation for this package was to factorize out that valuable component, so as to be able to build other powerful code generation
tools independently from go-swagger. Further improvements may be added independently.

The original version may be found [here](https://github.com/go-swagger/go-swagger/tree/master/generator/support.go#L129).

There are a few differences though:

* this version is intended to be very generic and has been stripped from any target-dependent settings
* all it knows is that is that it should use a templates Repository to render (primarily) some go code
* rendering may be called concurrently
* it provides a helper to generate go modules with their own go.mod
* it maintains it own set of optional settings
