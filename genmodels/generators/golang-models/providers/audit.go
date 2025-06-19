package providers

import (
	"github.com/fredbi/core/genmodels/generators/internal/audit"
	"github.com/fredbi/core/jsonschema/analyzers/structural"
)

type auditFunc = func(structural.AuditAction, string)

func noaudit(_ structural.AuditAction, _ string) {}

func (p *NameProvider) prepareAuditSchema(
	analyzed *structural.AnalyzedSchema,
) (auditFunc, func()) {
	audit := structural.AuditTrailEntry{
		Originator: audit.Originator(1),
	}
	didSomething := false
	did := noaudit
	deferred := func() {}

	if analyzed == nil {
		return did, deferred
	}

	if p.auditor != nil {
		// prepare for logging our action on return: post an audit entry into the original schema
		deferred = func() {
			if !didSomething {
				return
			}

			p.auditor.LogAudit(*analyzed, audit)
		}

		did = func(action structural.AuditAction, description string) {
			// describe the action performed
			didSomething = true
			audit.Action = action
			audit.Description = description
		}
	}

	return did, deferred
}

func (p *NameProvider) prepareAuditPackage(
	analyzed *structural.AnalyzedPackage,
) (func(structural.AuditAction, string), func()) {
	audit := structural.AuditTrailEntry{
		Originator: audit.Originator(1),
	}
	didSomething := false
	did := noaudit
	deferred := func() {}

	if analyzed == nil {
		return did, deferred
	}

	if p.auditor != nil {
		// prepare for logging our action on return: post an audit entry into the original schema
		deferred = func() {
			if !didSomething {
				return
			}

			p.auditor.LogAuditPackage(*analyzed, audit)
		}

		did = func(action structural.AuditAction, description string) {
			// describe the action performed
			didSomething = true
			audit.Action = action
			audit.Description = description
		}
	}

	return did, deferred
}
