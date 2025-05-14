# Templates repository

The templates repository is a cache for collecting golang text templates.

Features:

* unique template key built on top of a tree of template files
* caches compiled templates from assets on a file system, possibly embedded
* automatically resolves dependencies
* supports overlays: templates may be overridden from another source
* supports protected templates, to guard sensitive templates against unwanted overrides
* concurrency-safe
* generate documentation for your templates from comments in source


Functionality exposed as a separate module:

* [default functions map for golang code generation](funcmaps/golang/README.md)
