package structural

// AuditTrail
type AuditTrail struct {
	OriginalName string
	NameOverride string // x-go-name
}

func (t AuditTrail) Report() {
	// TODO
}
