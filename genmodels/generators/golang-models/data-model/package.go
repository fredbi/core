package model

import "github.com/fredbi/core/jsonschema/analyzers/structural"

// TargetPackage holds the data model for templates applied once per package (e.g. doc.go, possibly others)
type TargetPackage struct {
	GenPackageOptions
	GenPackageTemplateOptions
	LocationInfo

	Index  int64
	Source *structural.AnalyzedPackage

	// TODO: capture metadata about package
}

type GenPackageOptions struct{}

type GenPackageTemplateOptions struct {
	*GenOptions
	PkgTargetCodeFlags
}

type PkgTargetCodeFlags struct {
	NeedsDoc    bool
	NeedsReadme bool
}
