package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"aethel-core/internal/domain"
)

// GovernanceService mediates access to the audit ledger for query and verification.
type GovernanceService struct {
	audit domain.AuditRepository
}

func NewGovernanceService(audit domain.AuditRepository) *GovernanceService {
	return &GovernanceService{audit: audit}
}

// QueryAuditLog returns a paginated slice of audit entries for the given time window.
func (s *GovernanceService) QueryAuditLog(ctx context.Context, orgID uuid.UUID, from, to time.Time, page domain.Page) ([]domain.AuditEntry, error) {
	return s.audit.Query(ctx, orgID, from, to, page)
}

// VerifyChain re-validates the checksum chain for the given time window.
func (s *GovernanceService) VerifyChain(ctx context.Context, orgID uuid.UUID, from, to time.Time) (*domain.ChainVerificationResult, error) {
	return s.audit.VerifyChain(ctx, orgID, from, to)
}
