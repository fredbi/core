package differ

type Change struct {
	severity           Severity
	category           Category
	validationCategory ValidationCategory
	difftype           Type
	context            struct{} // TODO: Document context
}

type Result struct {
	changes []Change
}
