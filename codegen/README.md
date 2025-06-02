# codegen

A set of tools to build code generation applications.

## [genapp](./genapp/README.md)

A general-purpose type to generate go code.

It stitches together a [`templates-repo`](./templates-repo/README.md),
the associated template [`funcmap`](./funcmaps/README.md) and knows how
to render the passed data items.

## [templates-repo](./templates-repo/README.md)

A repository to cache golang `text/template` items and resolve dependencies.

## [funcmaps](./funcmaps/README.md)

A collection of templates funcmap that are used by go-openapi code generators.

## [gentesting](./gentesting/README.md)

A testing utility to assert the behavior of a dynamically generated package.

It wraps a package dynamically into a go plugin object and exposes an object to run
assertions against this plugin.

## [settings](./settings/README.md)

Utilities to manage and document settings used by code generation tools.

It produces markdown documentation, a CLI flags setup from settings.
