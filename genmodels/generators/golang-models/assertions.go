package models

import "fmt"

func assertNoPathEscapeError(err error, pth string) {
	if err != nil {
		panic(fmt.Errorf(
			"package paths provided by the structural.Analyzer are assumed to be sanitized, clean URLs, but got: %q: %w: %w",
			pth,
			err,
			ErrInternal,
		))
	}
}
