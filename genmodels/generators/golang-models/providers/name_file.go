package providers

import (
	"strconv"
	"strings"

	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

/*
type fileNamespaces map[string]fileNamespace

func MakeFileNamespaces() fileNamespaces {
	return make(map[string]fileNamespace)
}

func (n fileNamespaces) CheckNoConflictInPath(path string, file string) bool {
	namespace, ok := n[path]
	if !ok {
		return true
	}

	return namespace.CheckNoConflict(structural.Ident(file))
}

var _ structural.Namespace = fileNamespace{}

type fileNamespace struct {
	path  string
	files map[string]struct{}
}

func MakeFileNamespace(path string) fileNamespace {
	return fileNamespace{
		path:  path,
		files: make(map[string]struct{}),
	}
}

func (f fileNamespace) Path() string {
	return f.path
}

func (f fileNamespace) CheckNoConflict(ident structural.Ident) (ok bool) {
	_, ok = f.files[string(ident)]

	return !ok
}

// backtracking is problematic because the file naming is performed on the fly without
func (f fileNamespace) Backtrack(resolved structural.ConflictMeta) (ok bool) {
	if !f.CheckNoConflict(resolved.Ident) {
		return false
	}

	oldFile := resolved.ID.String()
	_, found := f.files[oldFile]
	if !found {
		return false
	}

	// swap file names for the pre-existing entry
	newFile := string(resolved.Ident)
	delete(f.files, oldFile)
	f.files[newFile] = struct{}{}

	return true
}

func (f fileNamespace) Meta(ident structural.Ident) (structural.ConflictMeta, bool) {
	_, ok := f.files[string(ident)]

	return structural.ConflictMeta{}, ok
}
func (f *fileNamespace) Set(file string) (ok bool) {
	_, ok = f.files[file]
	if ok {
		return false
	}

	f.files[file] = struct{}{}

	return true
}
*/

// FileName produces a source file name to hold model code.
//
// It is possible to override a generated file name using "x-go-file-name".
//
// FileName produces legit, idiomatic file names:
//
// Examples:
//
//   - xyz_unix gets rewritten
//   - xyz_test gets rewritten
//   - Abc XYZ becomes abc_xyz
func (p NameProvider) FileName(name string, analyzed structural.AnalyzedSchema) string {
	const directive = "x-go-file-name"
	pth := analyzed.Path()

	if ext, isUserDefined := analyzed.GetExtension(directive); isUserDefined {
		goFile := ext.(string)

		if p.isFileConflict(goFile, pth) {
			return p.deconflictsFile(goFile, pth)
		}

		p.registerFile(goFile, pth)

		return goFile
	}

	goFile := p.mangler.ToGoFileName(name)
	if p.isFileConflict(goFile, pth) {
		return p.deconflictsFile(goFile, pth)
	}
	p.registerFile(goFile, pth)

	return goFile
}

// FileNameForTest produces a source file name to hold test code.
// TODO: like for package aliases, build a type that implements structural.Namespace for files.
func (p NameProvider) FileNameForTest(name string, analyzed structural.AnalyzedSchema) string {
	var suffix string
	if withoutTestSuffix, isTestFile := strings.CutSuffix(name, "_test"); isTestFile {
		name = withoutTestSuffix
		suffix = "_test"
	}
	pth := analyzed.Path()

	goFile := p.mangler.ToGoFileName(name) + suffix
	if p.isFileConflict(goFile, pth) {
		return p.deconflictsFile(goFile, pth)
	}
	p.registerFile(goFile, pth)

	return goFile
}

func (p NameProvider) registerFile(name, pth string) {
	namespace, ok := p.files[pth]
	if !ok {
		namespace = make(map[string]struct{})
	}

	namespace[name] = struct{}{}

	p.files[pth] = namespace
}

// isFileConflict detects if the file name we are about to generate for this artifact
func (p NameProvider) isFileConflict(name, pth string) bool {
	namespace, ok := p.files[pth]
	if !ok {
		return false
	}
	_, alreadyExists := namespace[name]

	return alreadyExists
}

// deconflictsFile finds a deconflicted file name.
//
// The strategy to deconflict file names is simplistic:
//
// "object A" and "Object_a" identifiers would produce the same file target: object_a.
//
// The first would remain "object_a" and the next found on will be named "object_a_2".
func (p NameProvider) deconflictsFile(name, pth string) string {
	var suffix string

	if withoutTestSuffix, isTestFile := strings.CutSuffix(name, "_test"); isTestFile {
		name = withoutTestSuffix
		suffix = "_test"
	}

	for i := 1; ; i++ {
		attempt := name + "_" + strconv.Itoa(i) + suffix
		goFile := p.mangler.ToGoFileName(attempt)
		if p.isFileConflict(goFile, pth) {
			continue
		}

		p.registerFile(goFile, pth)

		return goFile
	}
}
