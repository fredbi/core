package funcmaps

import (
	"text/template"

	"github.com/fredbi/core/swag/typeutils"
)

// Merge [template.FuncMap] s into the target [template.FuncMap].
func Merge(target template.FuncMap, merged ...template.FuncMap) template.FuncMap {
	return typeutils.MergeMaps(target, merged...)
}
