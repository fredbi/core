// Package funcmaps propose different language-oriented funcmaps.
//
// Such sets of functions are used as convenient defaults for code-generation and doc-generation tools.
package funcmaps

import (
	"maps"
	"text/template"
)

func Merge(fmaps ...template.FuncMap) template.FuncMap {
	if len(fmaps) == 0 {
		return nil
	}

	result := fmaps[0]
	for _, fmap := range fmaps[1:] {
		maps.Copy(result, fmap)
	}

	return result
}
