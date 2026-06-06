package service

import (
	"testing"

	"github.com/google/uuid"

	"aethel-core/internal/domain"
)

// BenchmarkRoutingRuleEngine benchmarks evaluateRules with 1000 rules
// (999 non-matching active rules followed by 1 matching rule at the end — worst case).
func BenchmarkRoutingRuleEngine(b *testing.B) {
	// Build a dispatch that only the last rule will match.
	docTypeID := uuid.New()
	deptID := uuid.New()
	userID := uuid.New()

	dispatch := &domain.Dispatch{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Direction:      domain.DirectionInbound,
		DocumentTypeID: docTypeID,
		PriorityLevel:  domain.PriorityRoutine,
		SenderName:     "Test Sender",
	}

	// 999 rules that do not match (DIRECTION == OUTBOUND, but dispatch is INBOUND).
	rules := make([]domain.RoutingRule, 0, 1000)
	for i := 0; i < 999; i++ {
		rules = append(rules, domain.RoutingRule{
			ID:           uuid.New(),
			Name:         "non-matching-rule",
			PriorityOrder: i,
			IsActive:     true,
			Conditions: []domain.RuleCondition{
				{
					ID:         uuid.New(),
					FieldName:  "DIRECTION",
					Operator:   "EQUALS",
					MatchValue: "OUTBOUND", // dispatch is INBOUND — never matches
				},
			},
			Destinations: []domain.RuleDestination{
				{
					ID:           uuid.New(),
					DepartmentID: &deptID,
				},
			},
		})
	}

	// Rule 1000 — matches on DIRECTION == INBOUND.
	rules = append(rules, domain.RoutingRule{
		ID:            uuid.New(),
		Name:          "matching-rule",
		PriorityOrder: 999,
		IsActive:      true,
		Conditions: []domain.RuleCondition{
			{
				ID:         uuid.New(),
				FieldName:  "DIRECTION",
				Operator:   "EQUALS",
				MatchValue: "INBOUND",
			},
		},
		Destinations: []domain.RuleDestination{
			{
				ID:     uuid.New(),
				UserID: &userID,
			},
		},
	})

	// Wire up a minimal DispatchService — the rule engine only uses the receiver,
	// no repo calls occur inside evaluateRules.
	svc := &DispatchService{}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dest := svc.evaluateRules(dispatch, rules)
		if dest == nil {
			b.Fatal("expected a destination — rule matching broken")
		}
	}
}

// BenchmarkRoutingRuleEngine_NoMatch benchmarks the full scan with zero matches
// (worst-case for rules that never match — all 1000 rules checked every time).
func BenchmarkRoutingRuleEngine_NoMatch(b *testing.B) {
	deptID := uuid.New()
	dispatch := &domain.Dispatch{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Direction:      domain.DirectionInbound,
		DocumentTypeID: uuid.New(),
		PriorityLevel:  domain.PriorityRoutine,
	}

	rules := make([]domain.RoutingRule, 1000)
	for i := range rules {
		rules[i] = domain.RoutingRule{
			ID:            uuid.New(),
			Name:          "miss",
			PriorityOrder: i,
			IsActive:      true,
			Conditions: []domain.RuleCondition{
				{
					ID:         uuid.New(),
					FieldName:  "DIRECTION",
					Operator:   "EQUALS",
					MatchValue: "OUTBOUND",
				},
			},
			Destinations: []domain.RuleDestination{{ID: uuid.New(), DepartmentID: &deptID}},
		}
	}

	svc := &DispatchService{}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = svc.evaluateRules(dispatch, rules)
	}
}
