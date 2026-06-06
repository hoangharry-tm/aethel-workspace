package repos

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"aethel-core/internal/database"
	"aethel-core/internal/domain"
)

type EscalationRuleRepo struct {
	db *sql.DB
	q  *database.QueryRegistry
}

func NewEscalationRuleRepo(db *sql.DB, q *database.QueryRegistry) *EscalationRuleRepo {
	return &EscalationRuleRepo{db: db, q: q}
}

func (r *EscalationRuleRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.EscalationRule, error) {
	rows, err := r.q.Get("escalation_rules.list").Stmt.QueryContext(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("escalation_rule_repo: list: %w", err)
	}
	defer rows.Close()

	var result []domain.EscalationRule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, rule)
	}
	return result, rows.Err()
}

func (r *EscalationRuleRepo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*domain.EscalationRule, error) {
	row := r.q.Get("escalation_rules.fetch_by_id").Stmt.QueryRowContext(ctx, id, orgID)
	rule, err := scanRuleRow(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("escalation_rule_repo: get by id: %w", err)
	}
	return rule, nil
}

func (r *EscalationRuleRepo) Create(ctx context.Context, rule *domain.EscalationRule) error {
	row := r.q.Get("escalation_rules.create").Stmt.QueryRowContext(ctx,
		rule.ID,
		rule.OrganizationID,
		rule.Name,
		rule.TriggerHours,
		rule.EscalateToUserID,
		rule.EscalateToRole,
		rule.IsActive,
	)
	created, err := scanRuleRow(row)
	if err != nil {
		return fmt.Errorf("escalation_rule_repo: create: %w", err)
	}
	*rule = *created
	return nil
}

func (r *EscalationRuleRepo) Update(ctx context.Context, rule *domain.EscalationRule) error {
	row := r.q.Get("escalation_rules.update").Stmt.QueryRowContext(ctx,
		rule.ID,
		rule.Name,
		rule.TriggerHours,
		rule.EscalateToUserID,
		rule.EscalateToRole,
		rule.IsActive,
		rule.OrganizationID,
	)
	updated, err := scanRuleRow(row)
	if err == sql.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("escalation_rule_repo: update: %w", err)
	}
	*rule = *updated
	return nil
}

// rowScanner matches both *sql.Row and *sql.Rows for shared scanning logic.
type rowScanner interface {
	Scan(dest ...any) error
}

type ruleRow struct {
	id               uuid.UUID
	orgID            uuid.UUID
	name             string
	triggerHours     int
	escalateToUserID uuid.NullUUID
	escalateToRole   sql.NullString
	isActive         bool
	createdAt        time.Time
	updatedAt        time.Time
}

func scanFields(r rowScanner) (*ruleRow, error) {
	var row ruleRow
	if err := r.Scan(
		&row.id,
		&row.orgID,
		&row.name,
		&row.triggerHours,
		&row.escalateToUserID,
		&row.escalateToRole,
		&row.isActive,
		&row.createdAt,
		&row.updatedAt,
	); err != nil {
		return nil, err
	}
	return &row, nil
}

func toRule(row *ruleRow) domain.EscalationRule {
	rule := domain.EscalationRule{
		ID:             row.id,
		OrganizationID: row.orgID,
		Name:           row.name,
		TriggerHours:   row.triggerHours,
		IsActive:       row.isActive,
		CreatedAt:      row.createdAt,
		UpdatedAt:      row.updatedAt,
	}
	if row.escalateToUserID.Valid {
		rule.EscalateToUserID = &row.escalateToUserID.UUID
	}
	if row.escalateToRole.Valid {
		rule.EscalateToRole = &row.escalateToRole.String
	}
	return rule
}

func scanRule(rows *sql.Rows) (domain.EscalationRule, error) {
	row, err := scanFields(rows)
	if err != nil {
		return domain.EscalationRule{}, fmt.Errorf("escalation_rule_repo: scan: %w", err)
	}
	return toRule(row), nil
}

func scanRuleRow(sqlRow *sql.Row) (*domain.EscalationRule, error) {
	row, err := scanFields(sqlRow)
	if err != nil {
		return nil, err
	}
	rule := toRule(row)
	return &rule, nil
}
