package handlers

import (
	"aethel-core/internal/audit"
	"aethel-core/internal/domain"
	"aethel-core/internal/transport"
)

// AdminDeps groups the repository dependencies needed by AdminHandler.
type AdminDeps struct {
	Users        domain.UserRepository
	DocTypes     domain.DocumentTypeRepository
	RoutingRules domain.RoutingRuleRepository
	EscRules     domain.EscalationRuleRepository
	Audit        audit.Writer
}

// NotificationDeps groups the dependencies needed by NotificationHandler.
type NotificationDeps struct {
	Repo      domain.NotificationRepository
	SSEBroker *transport.SSEBroker
}
