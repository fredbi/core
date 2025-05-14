package golang

import (
	"text/template"

	"github.com/go-openapi/swag"
)

var (
	assets             map[string][]byte
	protectedTemplates map[string]bool

	// FuncMapFunc yields a map with all functions for templates
	FuncMapFunc func(*LanguageOpts) template.FuncMap

	//templates *Repository

	docFormat map[string]string
)

func initTemplateRepo() {
	FuncMapFunc = DefaultFuncMap

	// this makes the ToGoName func behave with the special
	// prefixing rule above
	swag.GoNamePrefixFunc = prefixForName

	//	assets = defaultAssets()
	//protectedTemplates = defaultProtectedTemplates()
	//templates = New(WithFuncMap(FuncMapFunc(DefaultLanguageFunc())))

	docFormat = map[string]string{
		"binary": "binary (byte stream)",
		"byte":   "byte (base64 string)",
	}
}
