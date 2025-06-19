package providers

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

// UniquePath yields the key that should be considered unique for package paths.
//
// For go generation, we are interested in the uniqueness of the paths transformed by ToGoPackagePath().
func (p *NameProvider) UniquePath(pth string) structural.Ident {
	return structural.Ident(p.mangler.ToGoPackagePath(pth))
}

// NamePackage knows how to determine the relative go package pth for a schema, when called back by the analyzer.
//
// The [structural.Analyzer] calls this whenever it visits a package node, that is a namespace for schemas.
//
// The result is a relative slash-separated path.
//
// Example: generated/models/go-folder
//
// It rewrites names to get legit, idiomatic go package names:
//
// * x_test gets rewritten as xtest
// * x/v2 gets rewritten as xv2
// * Abc gets rewritten as abc
// * computeService gets rewritten as compute-service
// * compute_service gets rewritten as compute-service
//
// TODO: search go doc for constraints on legit chars, unicode etc
func (p *NameProvider) NamePackage(
	pth string,
	analyzed structural.AnalyzedPackage,
) (goPkg string, err error) {
	const directive = "x-go-package"
	did, logAudit := p.prepareAuditPackage(&analyzed)
	defer logAudit()

	if p.marker != nil {
		// document the package with the original pth name
		defer func() {
			if goPkg == pth {
				return
			}

			mark := make(structural.Extensions, 1)
			mark.Add("x-go-original-pth", analyzed.Path())
			p.marker.MarkPackage(analyzed, mark)
		}()
	}

	if ext, isUserDefined := analyzed.GetExtension(directive); isUserDefined {
		goPkg = ext.(string)

		did(structural.AuditActionRenamePackage, fmt.Sprintf(
			"applied directive %s to rename package name %q to %q", directive, pth, goPkg,
		))

		return goPkg, nil
	}

	goPkg = p.mangler.ToGoPackagePath(pth)
	did(structural.AuditActionNamePackage, fmt.Sprintf(
		"applied mangler ToGoPackagePath to transform %q into %q", pth, goPkg,
	))

	return goPkg, nil
}

// DeconflictPath deconflicts a package path.
//
// Conflicts in path are usually introduced by the use of special characters, or case-sensitive identifiers.
//
// Since the path transforms removes these differences, unique JSON paths may point to the same "go package" location.
//
// Example:
//
// definitions:
//
//	Here:
//	  Models:
//	    parent: {}
//	here:
//	  models:
//	    child: {}
//	  Models:
//	    child: {}
//
// This structure produces the same package path 3 times: "here/models"
func (p *NameProvider) DeconflictPath(
	pth string,
	namespace structural.Namespace,
) (goPkg string, err error) {
	existing, ok := namespace.Meta(p.UniquePath(pth))
	if !ok {
		// TODO code assertion - we don't want to handle internal errors
		return "", fmt.Errorf(
			"invalid conflict detection: %w",
			ErrInternal,
		)
	}

	did, logAudit := p.prepareAuditPackage(existing.Package)
	defer logAudit()

	// find a common path where path strings match exactly (with case or other ignored character)
	assertConflictMetaMustHavePackage(existing.Package, pth)
	originalConflicting := existing.Package.OriginalName()
	commonPrefix := longestIdentical(pth, originalConflicting)

	// search last /
	lastSlash := strings.LastIndex(commonPrefix, "/")
	if lastSlash > 0 {
		commonPrefix = commonPrefix[:lastSlash-1]
	}

	// determine the longest "common path" corresponding to both packages.
	baseCommonPath := p.mangler.ToGoPackagePath(commonPrefix)

	// mangle the remaining part
	newSuffixPath := p.mangler.ToGoPackagePath(pth[lastSlash+1:])

	for index := range maxAttempts {
		newSuffixPath += strconv.Itoa(index)
		goPkg = path.Join(baseCommonPath, newSuffixPath)
		if !namespace.CheckNoConflict(p.UniquePath(goPkg)) {
			continue
		}

		did(structural.AuditActionRenamePackage, fmt.Sprintf(
			"deconflicted package path by adding to %q the index %d, then applied mangler ToGoPackagePath: %q",
			pth,
			index,
			goPkg,
		))

		return goPkg, nil
	}

	assertMustDeconflictUsingIndex(pth)

	// never reach this, we panic before
	return "", nil
}

// PackageShortName provides the package name to be used in the "package" statement.
//
// A [structural.analyzedSchema] is provided for context, but is not required.
//
// Examples:
//
//   - generated/models/go-folder -> "folder"
//   - generated/models/go-folder/v2 -> "folder"
func (p *NameProvider) PackageShortName(pth string, analyzed ...structural.AnalyzedSchema) string {
	_ = analyzed

	return p.mangler.ToGoPackageName(pth)
}

// PackageFullName returns the fully qualified package name, to be used in imports.
//
// A [structural.analyzedSchema] is provided for context, but is not required.
//
// Example:
//
//   - generated/models/go-folder -> "github.com/fredbi/core/genmodels/generated/models/go-folder"
func (p *NameProvider) PackageFullName(pth string, analyzed ...structural.AnalyzedSchema) string {
	_ = analyzed

	return path.Join(p.baseImportPath, p.mangler.ToGoPackagePath(pth))
}

// PackageAlias forms an alias for a package name, given the number of parts to use in addition
// to the last significant part (which gives the short name).
//
// When the number of parts is zero or less, the result is the same as [NameProvider.PackageShortName].
//
// Example:
//
//   - PackageAlias("generated/models/go-folder/v2",0) -> "folder"
//   - PackageAlias("generated/models/go-folder/v2",1) -> "folderv2"
//   - PackageAlias("generated/models/go-folder/v2",2) -> "gofolderv2"
func (p *NameProvider) PackageAlias(
	pth string,
	parts int,
	analyzed ...structural.AnalyzedSchema,
) string {
	_ = analyzed

	return p.mangler.ToGoPackageAlias(pth, parts)
}

// DeconflictAlias deconflicts a package alias in a namespace of packages provider by the data model.
//
// The first deconfliction strategy concatenates parts from the package path that would be ignored by the short name.
//
// Example:
//
//   - "generated/models/go-folder/v2" -> "folder"
//   - "generated/models/py-folder" -> "folder"! (enter DeconflictAlias)
//
// Yields:
//
//   - "generated/models/py-folder" -> "folder"! (enter DeconflictAlias) -> "pyfolder"
//
// We try to avoid joining to many parts this way (up to 3). If the conflict cannot be resolved [DeconflictAlias] falls
// back on an alternate strategy, that backtracks on a previous aliasing decision and try to find a better balance.
//
// Again, this may fail, so the strategy of last resort is to add a numerical index to the alias, until we
// find a high enough number to produce a different alias.
//
// With that strategy we would have:
//
//   - "folder"
//   - "folder2"
func (p *NameProvider) DeconflictAlias(
	name string,
	namespace structural.Namespace,
) (goName string, err error) {
	// 1. strategy based on folding parts
	const maxNumParts = 3 // max number of attempts to fold a previous part of the package name to produce a distinctive alias
	meta, _ := namespace.Meta(structural.Ident(name))
	analyzed := meta.Package
	did, logAudit := p.prepareAuditPackage(analyzed)
	defer logAudit()

	for parts := range maxNumParts {
		attempt := p.PackageAlias(name, parts)
		if namespace.CheckNoConflict(structural.Ident(attempt)) {
			goName = attempt
			did(structural.AuditActionPackageInfo, fmt.Sprintf(
				"deconflicted package alias by joining package parts, got %q, then applied mangler ToGoAlias with %d parts: %q",
				name,
				parts,
				goName,
			))

			return goName, err
		}
	}

	// 2. strategy based on folding parts, with backtracing
	backtrackable, useBacktrackStragegy := namespace.(structural.BacktrackableNamespace)
	// [model.ImportsMap] implements a [structural.BacktrackableNamespace]
	if useBacktrackStragegy {
		// we could not deconflict with 3 parts. Try with backtracking on a previous aliasing decision:
		// best to balance parts-based aliasing on 2 packages than having one with too many parts.
		//
		// Example:
		//
		// In the following sequence of packages, the import namespace is resolved with aliases like:
		//
		//  - "github.com/owner/repo/pkg/go/py-structs" -> "structs"
		//  - "github.com/owner/repo/pkg/go/go-gostructs" -> "gostructs"
		//  - "github.com/owner/repo/pkg/go/gogostructs" -> "gogostructs"
		//  - "github.com/owner/repo/pkg/go/gogogostructs" -> "gogogostructs"
		//  - "github.com/owner/repo/pkg/go/go-go-go-structs" -> "structs" (enter DeconflictAlias) -> "gostructs"! (0) -> "gogostructs"! (1) -> "gogogostructs"! (2) (stop)
		//
		// fail to pass the above deconfliction, as concatenating parts will always hit a conflict:
		// "structs", "gostructs", "gogostructs", "gogogostructs"
		//
		// The previous attempt is not patient enough to wait until a deconflicted "pkggogogostruct" alias emerges (because
		// it is likely a long and awkward identifier).
		//
		// After 3 iterations, we thus decide to backtrack on the first package found as conflicting: "structs" an alias it as "pystructs"
		//
		// So we have:
		//
		//  - "github.com/owner/repo/pkg/go/py-structs" -> "pystructs"
		//  - "github.com/owner/repo/pkg/go-go-go-structs" -> "structs"
		done := false
		existing, _ := namespace.Meta(structural.Ident(name))

		for parts := range maxNumParts {
			attempt := p.PackageAlias(existing.Name, parts)
			if namespace.CheckNoConflict(structural.Ident(attempt)) {
				resolved := existing
				existing.Ident = structural.Ident(attempt)
				done = backtrackable.Backtrack(resolved)
				if done {
					goName = name

					did(structural.AuditActionPackageInfo, fmt.Sprintf(
						"deconflicted package alias by backtracing on a previous aliasing for package %q, now realiased to %q. Current alias: %q",
						existing.Name,
						attempt,
						goName,
					))

					break
				}
			}
		}

		if done {
			return goName, err
		}
	}

	// 3. strategy based on indexing

	// still not a good result. As a last resort, simply add a number. This will eventually work, with a sufficiently high index.
	//
	// Example:
	//
	// We may get there with the following sequence:
	//
	//  - "github.com/owner/repo/pkg/go/go-go-go-structs" -> "structs"
	//  - "github.com/owner/repo/pkg/go/gostructs" -> "gostructs"
	//  - "github.com/owner/repo/pkg/go/go-gogostructs" -> "gogostructs"
	//  - "github.com/owner/repo/pkg/go/gogogostructs" -> "gogogostructs"
	//  - "github.com/owner/repo/pkg/go-go-go-structs" -> "structs"! (enter DeconflictAlias) -> "gostructs"! (0) -> "gogostructs"! (1) -> "gogogostructs"! (2) (stop)
	//
	// Since the backtracking strategy is not recursive, if fails again to find a deconflicted solution in 3 iterations.
	// So we have:
	//
	//  - "github.com/owner/repo/pkg/go-go-go-structs" -> "structs2"
	for idx := range maxAttempts {
		goName = name + strconv.Itoa(idx+indexingStart)
		if namespace.CheckNoConflict(structural.Ident(goName)) {
			did(structural.AuditActionPackageInfo, fmt.Sprintf(
				"deconflicted package alias using degraded naming strategy, got %q then iterated index to %d: %q",
				name,
				idx,
				goName,
			))

			return goName, nil
		}
	}

	// unless we hit an internal error or bug, we never get there: panic
	assertMustDeconflictUsingIndex(name)

	// never get there: we want alias deconfliction to always complete
	return "", nil
}

func longestIdentical(s1, s2 string) string {
	r2 := []rune(s2)
	j := 0

	for i, c := range s1 {
		if i >= len(r2) || c != r2[i] {
			break
		}
		j++
	}

	return string(r2[:j])
}
