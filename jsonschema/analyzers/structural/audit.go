package structural

import (
	"github.com/fredbi/core/swag/typeutils"
)

// AuditTrailEntry
type AuditTrailEntry struct {
	Action      AuditAction
	Originator  string
	Description string
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

// auditTrail holds all audit trail entries logged by the analyzer action or callbacks.
type auditTrail struct {
	originalName string
	nameOverride string // x-go-name
	auditEntries []AuditTrailEntry
}

func (t auditTrail) Report() {
	// TODO
}

func (t auditTrail) OriginalName() string {
	return t.originalName
}

func (t auditTrail) HasNameOverride() bool {
	return t.nameOverride != ""
}
