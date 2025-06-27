package analyzers

// InformationReport contains the record of decisions made by the analysis tools.
type InformationReport struct {
	DecisionType string
	Decision     string
	Originator   string   // program/function signature
	Sources      []string // json pointers to source schema or config
}
