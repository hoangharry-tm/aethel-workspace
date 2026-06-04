package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PriorityLevel string

const (
	PriorityRoutine   PriorityLevel = "ROUTINE"
	PriorityPriority  PriorityLevel = "PRIORITY"
	PriorityImmediate PriorityLevel = "IMMEDIATE"
)

type DispatchStatus string

const (
	StatusPendingAssignment  DispatchStatus = "PENDING_ASSIGNMENT"
	StatusUnderReview        DispatchStatus = "UNDER_REVIEW"
	StatusInTransit          DispatchStatus = "IN_TRANSIT"
	StatusAttemptedDelivery  DispatchStatus = "ATTEMPTED_DELIVERY"
	StatusDelivered          DispatchStatus = "DELIVERED"
	StatusEscalated          DispatchStatus = "ESCALATED"
	StatusDispatched         DispatchStatus = "DISPATCHED"
	StatusRejected           DispatchStatus = "REJECTED"
)

type DispatchDirection string

const (
	DirectionInbound  DispatchDirection = "INBOUND"
	DirectionOutbound DispatchDirection = "OUTBOUND"
)

type Dispatch struct {
	ID                       uuid.UUID         `json:"id"`
	OrganizationID           uuid.UUID         `json:"organizationId"`
	TrackingNumber           string            `json:"trackingNumber"`
	Direction                DispatchDirection `json:"direction"`
	DocumentTypeID           uuid.UUID         `json:"documentTypeId"`
	SenderName               string            `json:"senderName"`
	SenderOrganization       *string           `json:"senderOrganization,omitempty"`
	RecipientName            *string           `json:"recipientName,omitempty"`
	RecipientOrganization    *string           `json:"recipientOrganization,omitempty"`
	RecipientAddress         *string           `json:"recipientAddress,omitempty"`
	AssignedUserID           *uuid.UUID        `json:"assignedUserId,omitempty"`
	AssignedDepartmentID     *uuid.UUID        `json:"assignedDepartmentId,omitempty"`
	SubmittedByUserID        uuid.UUID         `json:"submittedByUserId"`
	PriorityLevel            PriorityLevel     `json:"priorityLevel"`
	StatusState              DispatchStatus    `json:"statusState"`
	SubjectLine              *string           `json:"subjectLine,omitempty"`
	DeliveryMode             *string           `json:"deliveryMode,omitempty"`
	IsManuallyRouted         bool              `json:"isManuallyRouted"`
	OriginalSuggestedUserID  *uuid.UUID        `json:"originalSuggestedUserId,omitempty"`
	OverdueAt                *time.Time        `json:"overdueAt,omitempty"`
	AcknowledgedAt           *time.Time        `json:"acknowledgedAt,omitempty"`
	AcknowledgedByUserID     *uuid.UUID        `json:"acknowledgedByUserId,omitempty"`
	IsEscalated              bool              `json:"isEscalated"`
	CreatedAt                time.Time         `json:"createdAt"`
	UpdatedAt                time.Time         `json:"updatedAt"`
}

type DispatchEvent struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organizationId"`
	DispatchID     uuid.UUID  `json:"dispatchId"`
	EventType      string     `json:"eventType"`
	ActorUserID    *uuid.UUID `json:"actorUserId,omitempty"`
	FromUserID     *uuid.UUID `json:"fromUserId,omitempty"`
	ToUserID       *uuid.UUID `json:"toUserId,omitempty"`
	FromDeptID     *uuid.UUID `json:"fromDeptId,omitempty"`
	ToDeptID       *uuid.UUID `json:"toDeptId,omitempty"`
	Metadata       *string    `json:"metadata,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type RoutingRule struct {
	ID             uuid.UUID           `json:"id"`
	OrganizationID uuid.UUID           `json:"organizationId"`
	Name           string              `json:"name"`
	PriorityOrder  int                 `json:"priorityOrder"`
	IsActive       bool                `json:"isActive"`
	Conditions     []RuleCondition     `json:"conditions"`
	Destinations   []RuleDestination   `json:"destinations"`
	CreatedAt      time.Time           `json:"createdAt"`
	UpdatedAt      time.Time           `json:"updatedAt"`
}

type RuleCondition struct {
	ID            uuid.UUID `json:"id"`
	RoutingRuleID uuid.UUID `json:"routingRuleId"`
	FieldName     string    `json:"fieldName"`
	Operator      string    `json:"operator"`
	MatchValue    string    `json:"matchValue"`
}

type RuleDestination struct {
	ID             uuid.UUID  `json:"id"`
	RoutingRuleID  uuid.UUID  `json:"routingRuleId"`
	DepartmentID   *uuid.UUID `json:"departmentId,omitempty"`
	UserID         *uuid.UUID `json:"userId,omitempty"`
	PriorityOffset int        `json:"priorityOffset"`
}

type DocumentType struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	Name           string    `json:"name"`
	Code           string    `json:"code"`
	Description    *string   `json:"description,omitempty"`
	IsActive       bool      `json:"isActive"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type DispatchRepository interface {
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*Dispatch, error)
	GetByTrackingNumber(ctx context.Context, orgID uuid.UUID, trackingNumber string) (*Dispatch, error)
	ListInbox(ctx context.Context, orgID, deptID uuid.UUID, page Page) ([]Dispatch, error)
	ListOutbound(ctx context.Context, orgID uuid.UUID, page Page) ([]Dispatch, error)
	ListByUser(ctx context.Context, orgID, userID uuid.UUID, page Page) ([]Dispatch, error)
	ListUnassigned(ctx context.Context, orgID uuid.UUID, page Page) ([]Dispatch, error)
	Create(ctx context.Context, d *Dispatch) error
	UpdateStatus(ctx context.Context, orgID, id uuid.UUID, status DispatchStatus) error
	Assign(ctx context.Context, orgID, id uuid.UUID, userID, deptID *uuid.UUID) error
	Acknowledge(ctx context.Context, orgID, id uuid.UUID, byUserID uuid.UUID) error
	Escalate(ctx context.Context, orgID, id uuid.UUID) error
}

type DispatchEventRepository interface {
	Create(ctx context.Context, e *DispatchEvent) error
	ListByDispatch(ctx context.Context, orgID, dispatchID uuid.UUID) ([]DispatchEvent, error)
}

type RoutingRuleRepository interface {
	List(ctx context.Context, orgID uuid.UUID) ([]RoutingRule, error)
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*RoutingRule, error)
	Create(ctx context.Context, r *RoutingRule) error
	Update(ctx context.Context, r *RoutingRule) error
	Delete(ctx context.Context, orgID, id uuid.UUID) error
}

type DocumentTypeRepository interface {
	List(ctx context.Context, orgID uuid.UUID) ([]DocumentType, error)
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*DocumentType, error)
	Create(ctx context.Context, dt *DocumentType) error
	Update(ctx context.Context, dt *DocumentType) error
	Delete(ctx context.Context, orgID, id uuid.UUID) error
}
