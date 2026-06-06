package handlers

import (
	"aethel-core/internal/audit"
	"aethel-core/internal/domain"
)

// AdminDeps groups the repository dependencies needed by AdminHandler.
type AdminDeps struct {
	Users        domain.UserRepository
	DocTypes     domain.DocumentTypeRepository
	RoutingRules domain.RoutingRuleRepository
	EscRules     domain.EscalationRuleRepository
	Audit        audit.Writer
}
