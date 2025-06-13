package structural

// AuditTrail
type AuditTrail struct {
	originalName string
	nameOverride string // x-go-name
}

func (t AuditTrail) Report() {
	// TODO
}

func (t AuditTrail) OriginalName() string {
	return t.originalName
}

func (t AuditTrail) HasNameOverride() bool {
	return t.nameOverride != ""
}
