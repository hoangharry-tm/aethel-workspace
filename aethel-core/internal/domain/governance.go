package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AuditEventType string

const (
	AuditUserLogin                  AuditEventType = "USER_LOGIN"
	AuditUserLoginFailed            AuditEventType = "USER_LOGIN_FAILED"
	AuditUserLogout                 AuditEventType = "USER_LOGOUT"
	AuditSessionRevoked             AuditEventType = "SESSION_REVOKED"
	AuditPermissionDenied           AuditEventType = "PERMISSION_DENIED"
	AuditRBACElevationAttempt       AuditEventType = "RBAC_ELEVATION_ATTEMPT"
	AuditSecurityBreachAttempt      AuditEventType = "SECURITY_BREACH_ATTEMPT"
	AuditUnauthorizedAccessBypassed AuditEventType = "UNAUTHORIZED_ACCESS_BYPASSED"
	AuditDispatchCreated            AuditEventType = "DISPATCH_CREATED"
	AuditDispatchAssigned           AuditEventType = "DISPATCH_ASSIGNED"
	AuditDispatchDelivered          AuditEventType = "DISPATCH_DELIVERED"
	AuditGreenNoteAppended          AuditEventType = "GREEN_NOTE_APPENDED"
	AuditMinuteSheetApproved        AuditEventType = "MINUTE_SHEET_APPROVED"
	AuditAdminUserCreated           AuditEventType = "ADMIN_USER_CREATED"
	AuditAdminUserDeactivated       AuditEventType = "ADMIN_USER_DEACTIVATED"
	AuditAdminSettingsChanged       AuditEventType = "ADMIN_SETTINGS_CHANGED"
	AuditRoutingRuleModified        AuditEventType = "ROUTING_RULE_MODIFIED"
	AuditEscalationFired            AuditEventType = "ESCALATION_FIRED"
)

type AuditEntry struct {
	ID               int64          `json:"id"`
	OrganizationID   uuid.UUID      `json:"organizationId"`
	ActorUserID      *uuid.UUID     `json:"actorUserId,omitempty"`
	ActionEventType  AuditEventType `json:"actionEventType"`
	TargetResourceID *uuid.UUID     `json:"targetResourceId,omitempty"`
	TargetTable      *string        `json:"targetTable,omitempty"`
	IPAddress        *string        `json:"ipAddress,omitempty"`
	UserAgent        *string        `json:"userAgent,omitempty"`
	Metadata         *string        `json:"metadata,omitempty"`
	PreviousChecksum string         `json:"previousChecksum"`
	Checksum         string         `json:"checksum"`
	CreatedAt        time.Time      `json:"createdAt"`
}

type ChainVerificationResult struct {
	Valid      bool            `json:"valid"`
	TotalRows  int             `json:"totalRows"`
	BrokenAt   []BrokenLink    `json:"brokenAt,omitempty"`
}

type BrokenLink struct {
	EntryID          int64     `json:"entryId"`
	ComputedChecksum string    `json:"computedChecksum"`
	StoredChecksum   string    `json:"storedChecksum"`
	CreatedAt        time.Time `json:"createdAt"`
}

type EscalationRule struct {
	ID                 uuid.UUID  `json:"id"`
	OrganizationID     uuid.UUID  `json:"organizationId"`
	Name               string     `json:"name"`
	TriggerHours       int        `json:"triggerHours"`
	EscalateToUserID   *uuid.UUID `json:"escalateToUserId,omitempty"`
	EscalateToRole     *string    `json:"escalateToRole,omitempty"`
	IsActive           bool       `json:"isActive"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

type AuditRepository interface {
	Write(ctx context.Context, entry *AuditEntry) error
	Query(ctx context.Context, orgID uuid.UUID, from, to time.Time, page Page) ([]AuditEntry, error)
	VerifyChain(ctx context.Context, orgID uuid.UUID, from, to time.Time) (*ChainVerificationResult, error)
}

type EscalationRuleRepository interface {
	List(ctx context.Context, orgID uuid.UUID) ([]EscalationRule, error)
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*EscalationRule, error)
	Create(ctx context.Context, r *EscalationRule) error
	Update(ctx context.Context, r *EscalationRule) error
}
