package providers

import (
	"strconv"
	"strings"

	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

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
	namespace, ok := p.filesNamespaces[pth]
	if !ok {
		namespace = make(map[string]struct{})
	}

	namespace[name] = struct{}{}

	p.filesNamespaces[pth] = namespace

	return
}

// isFileConflict detects if the file name we are about to generate for this artifact
func (p NameProvider) isFileConflict(name, pth string) bool {
	namespace, ok := p.filesNamespaces[pth]
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
