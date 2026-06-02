package repos

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"aethel-core/internal/database"
	"aethel-core/internal/domain"
)

// RoutingRuleRepo implements domain.RoutingRuleRepository.
// Single-tenant: orgID parameters are accepted to satisfy the interface but not used in SQL.
type RoutingRuleRepo struct {
	db *sql.DB
	q  *database.QueryRegistry
}

func NewRoutingRuleRepo(db *sql.DB, q *database.QueryRegistry) *RoutingRuleRepo {
	return &RoutingRuleRepo{db: db, q: q}
}

// List returns all active routing rules with their conditions and destinations.
func (r *RoutingRuleRepo) List(ctx context.Context, _ uuid.UUID) ([]domain.RoutingRule, error) {
	rows, err := r.q.Get("routing_rule.list_active").Stmt.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRoutingRules(rows)
}

// GetByID returns a single routing rule with all conditions and destinations.
func (r *RoutingRuleRepo) GetByID(ctx context.Context, _ uuid.UUID, id uuid.UUID) (*domain.RoutingRule, error) {
	rows, err := r.q.Get("routing_rule.get_with_details").Stmt.QueryContext(ctx, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules, err := scanRoutingRules(rows)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return nil, domain.ErrNotFound
	}
	return &rules[0], nil
}

// Create inserts a new routing rule.
// Note: created_by_user_id is required NOT NULL in the DB; callers must set Rule.OrganizationID.
// Sprint 3-4 will wire full admin routing-rule CRUD — this is a placeholder implementation.
func (r *RoutingRuleRepo) Create(ctx context.Context, rule *domain.RoutingRule) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO routing_rules (id, organization_id, name, priority_order, is_active, created_by_user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now(), now())
	`, rule.ID, rule.OrganizationID, rule.Name, rule.PriorityOrder, rule.IsActive,
		uuid.Nil, // created_by_user_id placeholder — Sprint 3-4 wires the actor user ID
	)
	return err
}

// Update modifies an existing routing rule.
func (r *RoutingRuleRepo) Update(ctx context.Context, rule *domain.RoutingRule) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE routing_rules SET name = $2, priority_order = $3, is_active = $4, updated_at = now()
		WHERE id = $1
	`, rule.ID, rule.Name, rule.PriorityOrder, rule.IsActive)
	return err
}

// Delete removes a routing rule.
func (r *RoutingRuleRepo) Delete(ctx context.Context, _ uuid.UUID, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM routing_rules WHERE id = $1`, id)
	return err
}

// scanRoutingRules collapses the JOIN result (one row per condition×destination pair)
// into a slice of RoutingRule structs with their conditions and destinations deduplicated.
func scanRoutingRules(rows *sql.Rows) ([]domain.RoutingRule, error) {
	type ruleRow struct {
		ruleID     uuid.UUID
		orgID      uuid.UUID
		name       string
		priority   int
		isActive   bool
		createdAt  time.Time
		updatedAt  time.Time
		condID     uuid.NullUUID
		condType   sql.NullString
		condValue  sql.NullString
		condOp     sql.NullString
		destID     uuid.NullUUID
		destOrder  sql.NullInt32
		destUserID uuid.NullUUID
		destDeptID uuid.NullUUID
	}

	ruleMap := make(map[uuid.UUID]*domain.RoutingRule)
	ruleOrder := []uuid.UUID{}
	seenConds := make(map[uuid.UUID]bool)
	seenDests := make(map[uuid.UUID]bool)

	for rows.Next() {
		var rr ruleRow
		if err := rows.Scan(
			&rr.ruleID, &rr.orgID, &rr.name, &rr.priority, &rr.isActive,
			&rr.createdAt, &rr.updatedAt,
			&rr.condID, &rr.condType, &rr.condValue, &rr.condOp,
			&rr.destID, &rr.destOrder, &rr.destUserID, &rr.destDeptID,
		); err != nil {
			return nil, err
		}

		rule, exists := ruleMap[rr.ruleID]
		if !exists {
			rule = &domain.RoutingRule{
				ID:             rr.ruleID,
				OrganizationID: rr.orgID,
				Name:           rr.name,
				PriorityOrder:  rr.priority,
				IsActive:       rr.isActive,
				CreatedAt:      rr.createdAt,
				UpdatedAt:      rr.updatedAt,
			}
			ruleMap[rr.ruleID] = rule
			ruleOrder = append(ruleOrder, rr.ruleID)
		}

		if rr.condID.Valid && !seenConds[rr.condID.UUID] {
			seenConds[rr.condID.UUID] = true
			rule.Conditions = append(rule.Conditions, domain.RuleCondition{
				ID:            rr.condID.UUID,
				RoutingRuleID: rr.ruleID,
				FieldName:     rr.condType.String,
				MatchValue:    rr.condValue.String,
				Operator:      rr.condOp.String,
			})
		}

		if rr.destID.Valid && !seenDests[rr.destID.UUID] {
			seenDests[rr.destID.UUID] = true
			dest := domain.RuleDestination{
				ID:            rr.destID.UUID,
				RoutingRuleID: rr.ruleID,
				PriorityOffset: int(rr.destOrder.Int32),
			}
			if rr.destUserID.Valid {
				uid := rr.destUserID.UUID
				dest.UserID = &uid
			}
			if rr.destDeptID.Valid {
				did := rr.destDeptID.UUID
				dest.DepartmentID = &did
			}
			rule.Destinations = append(rule.Destinations, dest)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]domain.RoutingRule, 0, len(ruleOrder))
	for _, id := range ruleOrder {
		result = append(result, *ruleMap[id])
	}
	return result, nil
}
