package differ

import (
	"iter"
	"slices"
)

type Change struct {
	severity           Severity
	categories         CategoryMode
	validationCategory ValidationCategory
	difftype           Type
	context            struct{} // TODO: Document context: where is the change located
}

// Result of the differences analysis between two schemas.
type Result struct {
	changes []Change
}

type ChangesOption func(*changesOptions)

type changesOptions struct {
	orderBySeverityDesc     bool
	filterSeverityThreshold Severity
}

func (r Result) Changes(opts ...ChangesOption) iter.Seq[Change] {
	return slices.Values(r.changesWithOptions(opts))
}

func (r Result) changesWithOptions(opts []ChangesOption) []Change {
	if len(opts) == 0 {
		return r.changes
	}

	return nil // TODO
}
