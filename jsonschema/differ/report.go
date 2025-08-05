package differ

import "io"

// WriteReport knows how to render a diff [Result] as a human readable report.
func WriteReport(w io.Writer, r Result, opts ...ReportOption) error {
	return nil
}
