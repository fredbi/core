package model

import (
	"path/filepath"
)

type LocationInfo struct {
	BaseImportPath  string // the base generation target path (derived from TargetDir and TargetModuleRoot)
	Package         string // package short name (e.g. "models")
	PackageLocation string // relative path to the package (e.g. "models/subpackage/enums")
	FullPackage     string // fully qualified package name (e.g. "github.com/fredbi/core/models")
	File            string // file stem name without path or extension (e.g. "this_is_a_model")
	Template        string // the template to be used e.g. "models", "pkgdoc"
	Ext             string // the file extension. If not defined, the extension is ".go"
}

// FileName resolves the relative path to the file name.
func (p LocationInfo) FileName() string {
	ext := p.Ext
	if ext == "" {
		ext = ".go"
	}

	return filepath.Join(p.PackageLocation, p.File+ext)
}
