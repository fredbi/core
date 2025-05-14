package golang

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/go-openapi/swag"
	"github.com/kr/pretty"
)

// DefaultFuncMap yields a map with default functions for use in the templates.
// These are available in every template
func DefaultFuncMap(lang *LanguageOpts) template.FuncMap {
	f := sprig.TxtFuncMap()
	extra := template.FuncMap{
		"pascalize": pascalize,
		"camelize":  swag.ToJSONName,
		"varname":   lang.MangleVarName,
		"humanize":  swag.ToHumanNameLower,
		"snakize":   lang.MangleFileName,
		"toPackagePath": func(name string) string {
			return filepath.FromSlash(lang.ManglePackagePath(name, ""))
		},
		"toPackage": func(name string) string {
			return lang.ManglePackagePath(name, "")
		},
		"toPackageName": func(name string) string {
			return lang.ManglePackageName(name, "")
		},
		"dasherize":          swag.ToCommandName,
		"pluralizeFirstWord": pluralizeFirstWord,
		"json":               asJSON,
		"prettyjson":         asPrettyJSON,
		"hasInsecure": func(arg []string) bool {
			return swag.ContainsStringsCI(arg, "http") || swag.ContainsStringsCI(arg, "ws")
		},
		"hasSecure": func(arg []string) bool {
			return swag.ContainsStringsCI(arg, "https") || swag.ContainsStringsCI(arg, "wss")
		},
		"dropPackage":      dropPackage,
		"containsPkgStr":   containsPkgStr,
		"contains":         swag.ContainsStrings,
		"padSurround":      padSurround,
		"joinFilePath":     filepath.Join,
		"joinPath":         path.Join,
		"comment":          padComment,
		"blockcomment":     blockComment,
		"inspect":          pretty.Sprint,
		"cleanPath":        path.Clean,
		"mediaTypeName":    mediaMime,
		"mediaGoName":      mediaGoName,
		"arrayInitializer": lang.arrayInitializer,
		"hasPrefix":        strings.HasPrefix,
		"stringContains":   strings.Contains,
		"imports":          lang.imports,
		"dict":             dict,
		"isInteger":        isInteger,
		"escapeBackticks": func(arg string) string {
			return strings.ReplaceAll(arg, "`", "`+\"`\"+`")
		},
		/*
			"paramDocType": func(param GenParameter) string {
				return resolvedDocType(param.SwaggerType, param.SwaggerFormat, param.Child)
			},
			"headerDocType": func(header GenHeader) string {
				return resolvedDocType(header.SwaggerType, header.SwaggerFormat, header.Child)
			},
			"schemaDocType": func(in interface{}) string {
				switch schema := in.(type) {
				case GenSchema:
					return resolvedDocSchemaType(schema.SwaggerType, schema.SwaggerFormat, schema.Items)
				case *GenSchema:
					if schema == nil {
						return ""
					}
					return resolvedDocSchemaType(schema.SwaggerType, schema.SwaggerFormat, schema.Items)
				case GenDefinition:
					return resolvedDocSchemaType(schema.SwaggerType, schema.SwaggerFormat, schema.Items)
				case *GenDefinition:
					if schema == nil {
						return ""
					}
					return resolvedDocSchemaType(schema.SwaggerType, schema.SwaggerFormat, schema.Items)
				default:
					panic("dev error: schemaDocType should be called with GenSchema or GenDefinition")
				}
			},
			"schemaDocMapType": func(schema GenSchema) string {
				return resolvedDocElemType("object", schema.SwaggerFormat, &schema.resolvedType)
			},
		*/
		//"docCollectionFormat": resolvedDocCollectionFormat,
		"trimSpace":          strings.TrimSpace,
		"mdBlock":            markdownBlock, // markdown block
		"httpStatus":         httpStatus,
		"cleanupEnumVariant": cleanupEnumVariant,
		"gt0":                gt0,
		//"path":                errorPath,
		/*
			"cmdName": func(in interface{}) (string, error) {
				// builds the name of a CLI command for a single operation
				op, isOperation := in.(GenOperation)
				if !isOperation {
					ptr, ok := in.(*GenOperation)
					if !ok {
						return "", fmt.Errorf("cmdName should be called on a GenOperation, but got: %T", in)
					}
					op = *ptr
				}
				name := "Operation" + pascalize(op.Package) + pascalize(op.Name) + "Cmd"

				return name, nil // TODO
			},
			"cmdGroupName": func(in interface{}) (string, error) {
				// builds the name of a group of CLI commands
				opGroup, ok := in.(GenOperationGroup)
				if !ok {
					return "", fmt.Errorf("cmdGroupName should be called on a GenOperationGroup, but got: %T", in)
				}
				name := "GroupOfOperations" + pascalize(opGroup.Name) + "Cmd"

				return name, nil // TODO
			},
		*/
		"flagNameVar": func(in string) string {
			// builds a flag name variable in CLI commands
			return fmt.Sprintf("flag%sName", pascalize(in))
		},
		"flagValueVar": func(in string) string {
			// builds a flag value variable in CLI commands
			return fmt.Sprintf("flag%sValue", pascalize(in))
		},
		"flagDefaultVar": func(in string) string {
			// builds a flag default value variable in CLI commands
			return fmt.Sprintf("flag%sDefault", pascalize(in))
		},
		"flagModelVar": func(in string) string {
			// builds a flag model variable in CLI commands
			return fmt.Sprintf("flag%sModel", pascalize(in))
		},
		"flagDescriptionVar": func(in string) string {
			// builds a flag description variable in CLI commands
			return fmt.Sprintf("flag%sDescription", pascalize(in))
		},
	}

	for k, v := range extra {
		f[k] = v
	}

	return f
}
