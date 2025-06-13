package golang

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/fredbi/core/codegen/funcmaps"
	"github.com/fredbi/core/swag/mangling"
	"github.com/fredbi/core/swag/stringutils"

	"github.com/kr/pretty"
)

// DefaultFuncMap yields a map with default functions for use in the templates.
//
// The available functions are specifically biased toward golang code generation.
//
// These are made available to every template.
func DefaultFuncMap(opts ...Option) template.FuncMap {
	o := optionsWithDefaults(opts)
	mangler := mangling.New(o.manglerOptions...)

	return funcmaps.Merge(
		template.FuncMap{
			// extra builtins
			"gt0": gt0,
			"assert": func(msg string, assertion bool) (string, error) {
				if !assertion {
					return "", fmt.Errorf("%v: %w", msg, ErrTemplateAssertion)
				}
				return "", nil
			},
			// strings
			"hasPrefix":      strings.HasPrefix,
			"stringContains": strings.Contains,
			"contains":       stringutils.ContainsStrings,
			"containsCI":     stringutils.ContainsStringsCI,
			"padSurround":    padSurround,
			"pad":            func(n int) string { return strings.Repeat(" ", n) },
			"trimSpace":      strings.TrimSpace,
			// paths
			"joinFilePath": filepath.Join,
			"joinPath":     path.Join,
			"cleanPath":    path.Clean,
			"filename":     mangler.ToFileName,
			// common mangling
			"pascalize": mangler.Pascalize,
			"dasherize": mangler.Dasherize, // aka kebab-case
			"camelize":  mangler.ToJSONName,
			"humanize":  mangler.ToHumanNameLower,
			"snakize":   mangler.Snakize,
			// go mangling
			"varname":          mangler.ToVarName, // a go-safe version of camelize to build unexported go identifiers
			"toGoName":         mangler.ToGoName,  // a go-safe version of pascalize to build exported go identifiers
			"toPackagePath":    mangler.ToGoPackagePath,
			"toPackageName":    mangler.ToGoPackageName,
			"comment":          padComment,
			"blockcomment":     blockComment,
			"escapeBackticks":  escapeBackticks,
			"arrayInitializer": arrayInitializer,
			"imports":          generateImports,
			"dropPackage":      dropPackage, // drops the fully qualified package from a go identifier
			"containsPkgStr":   containsPkgStr,
			"isInteger":        isInteger,
			// APIs
			"httpStatus":         httpStatus,
			"hasInsecure":        hasInsecureScheme,
			"hasSecure":          hasSecureScheme,
			"mediaTypeName":      mediaMime,
			"pluralizeFirstWord": pluralizeFirstWord,
			// printing
			"json":       asJSON,
			"prettyjson": asPrettyJSON,
			"inspect":    pretty.Sprint,
			//
			"dict":    dict,
			"mdBlock": markdownBlock, // markdown block
		},
	)
}
