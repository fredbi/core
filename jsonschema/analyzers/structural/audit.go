package structural

// AuditTrail
type AuditTrail struct {
	originalName string
	nameOverride string // x-go-name
	auditEntries []AuditTrailEntry
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

type AuditAction uint8

const (
	AuditActionNone AuditAction = iota
	AuditActionRefactorSchema
	AuditActionRenameSchema
	AuditActionRenamePackage
	AuditActionDeconflictName
	AuditActionNameAnonymous
	AuditActionNameInfo
	AuditActionNamePackage
	AuditActionPackageInfo
	AuditActionMetadata
)

type AuditTrailEntry struct {
	Action      AuditAction
	Originator  string
	Description string
}
