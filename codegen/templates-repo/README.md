# Templates repository

The templates repository is a cache for collecting golang text templates.

Features:

* unique template key built on top of a tree of template files
* caches compiled templates from assets on a file system, possibly embedded
* automatically resolves dependencies
* supports overlays: templates may be overridden from another source
* generates documentation for your templates from comments in source
* is safe for a concurrent use


Functionality exposed as separate modules:

* [default functions map for golang code generation](https://github.com/fredbi/core/tree/master/codegen/funcmaps/golang)
* [golang code generation helper](https://github.com/fredbi/core/tree/master/codegen/genapp)

### Credits

This package has been built from the templates repo built for go-swagger.

The motivation for this package was to factorize out that valuable component, so as to be able to build other powerful code generation
tools independently from go-swagger. Further improvements may be added independently.

The original version may be found [here](https://github.com/go-swagger/go-swagger/tree/master/generator/template_repo.go).

There are a few differences though:

* this version of the Repository caches eagerly, not lazily.
  > This helps finding invalid templates earlier, and allows for concurrent use of the cache.

* this version of the Repository natively works with the [fs.FS] absraction of a read-only file system.
  > We use additonal helpers to support "overlay file systems", very much like "github.com/spf13/afero"
  > does it, but without adding that extra dependency.

* supports for protected templates and "allow override" setting has been removed 
  > The idea to guard sensitive templates against unwanted overrides has caused more confusion
  > than it helped. Besides, there are new ways to override, e.g. by providing an overlay file system
  > to resolve the sources, which would bypass template protection.

* template structure dump and documentation has been improved with docstrings parsed from comments
  in templates and a generated markdown documentation.
