package structural

import (
	"github.com/fredbi/core/swag/typeutils"
)

func (a *SchemaAnalyzer) LogAudit(s AnalyzedSchema, e AuditTrailEntry) {
	if e.Action == AuditActionNone {
		return
	}

	schema, found := a.index[s.id]
	if !found {
		return
	}

	schema.auditEntries = append(schema.auditEntries, e)
}

func (a *SchemaAnalyzer) LogAuditPackage(p AnalyzedPackage, e AuditTrailEntry) {
	if e.Action == AuditActionNone {
		return
	}

	pkg, found := a.pkgIndex[p.id]
	if !found {
		return
	}

	pkg.auditEntries = append(pkg.auditEntries, e)
}

func (a *SchemaAnalyzer) MarkSchema(s AnalyzedSchema, e Extensions) {
	if len(e) == 0 {
		return
	}

	schema, found := a.index[s.id]
	if !found {
		return
	}

	schema.extensions = typeutils.MergeMaps(schema.extensions, e)
}

func (a *SchemaAnalyzer) MarkPackage(s AnalyzedPackage, e Extensions) {
	if len(e) == 0 {
		return
	}

	schema, found := a.index[s.id]
	if !found {
		return
	}

	schema.extensions = typeutils.MergeMaps(schema.extensions, e)
}

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

// AuditAction categorizes the actions we want to audit.
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
