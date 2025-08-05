package differ

type ReportOption func(*reportOptions)

type ReportOutputMode uint8

const (
	ReportOutputTable ReportOutputMode = 1 << iota
	ReportOutputMarkdown
	ReportOutputHTML
)

type reportOptions struct {
	reportMode        ReportOutputMode
	severityThreshold Severity
}

func WithThreshold(minSeverity Severity) ReportOption {
	return func(o *reportOptions) {
		o.severityThreshold = minSeverity
	}
}
