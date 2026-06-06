package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"aethel-core/internal/domain"
)

// recordingAuditRepo satisfies domain.AuditRepository with in-memory storage.
type recordingAuditRepo struct {
	entries []domain.AuditEntry
}

func (m *recordingAuditRepo) Write(_ context.Context, entry *domain.AuditEntry) error {
	m.entries = append(m.entries, *entry)
	return nil
}

func (m *recordingAuditRepo) Query(_ context.Context, _ uuid.UUID, from, to time.Time, page domain.Page) ([]domain.AuditEntry, error) {
	var out []domain.AuditEntry
	for _, e := range m.entries {
		if !e.CreatedAt.Before(from) && e.CreatedAt.Before(to) {
			out = append(out, e)
		}
	}
	start := page.Offset
	if start > len(out) {
		return nil, nil
	}
	end := start + page.Limit
	if end > len(out) {
		end = len(out)
	}
	return out[start:end], nil
}

func (m *recordingAuditRepo) VerifyChain(_ context.Context, _ uuid.UUID, from, to time.Time) (*domain.ChainVerificationResult, error) {
	var entries []domain.AuditEntry
	for _, e := range m.entries {
		if !e.CreatedAt.Before(from) && e.CreatedAt.Before(to) {
			entries = append(entries, e)
		}
	}

	if len(entries) == 0 {
		return &domain.ChainVerificationResult{Valid: true, TotalRows: 0}, nil
	}

	var broken []domain.BrokenLink
	for i := 1; i < len(entries); i++ {
		expected := entries[i-1].Checksum
		if entries[i].PreviousChecksum != expected {
			broken = append(broken, domain.BrokenLink{
				EntryID:          entries[i].ID,
				ComputedChecksum: expected,
				StoredChecksum:   entries[i].PreviousChecksum,
				CreatedAt:        entries[i].CreatedAt,
			})
		}
	}
	return &domain.ChainVerificationResult{
		Valid:     len(broken) == 0,
		TotalRows: len(entries),
		BrokenAt:  broken,
	}, nil
}

func buildChain(n int, now time.Time) []domain.AuditEntry {
	entries := make([]domain.AuditEntry, n)
	for i := range entries {
		prev := ""
		if i > 0 {
			prev = entries[i-1].Checksum
		}
		entries[i] = domain.AuditEntry{
			ID:               int64(i + 1),
			ActionEventType:  domain.AuditDispatchCreated,
			PreviousChecksum: prev,
			Checksum:         fmt.Sprintf("checksum-%d", i),
			CreatedAt:        now.Add(time.Duration(i) * time.Second),
		}
	}
	return entries
}

func TestVerifyChain_CleanChain(t *testing.T) {
	now := time.Now()
	repo := &recordingAuditRepo{entries: buildChain(10, now)}
	svc := NewGovernanceService(repo)

	result, err := svc.VerifyChain(context.Background(), uuid.New(), now.Add(-time.Second), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected chain to be valid, brokenAt: %v", result.BrokenAt)
	}
	if len(result.BrokenAt) != 0 {
		t.Errorf("expected 0 broken links, got %d", len(result.BrokenAt))
	}
	if result.TotalRows != 10 {
		t.Errorf("expected 10 rows, got %d", result.TotalRows)
	}
}

func TestVerifyChain_BrokenAtRow5(t *testing.T) {
	now := time.Now()
	entries := buildChain(10, now)
	// Corrupt row 5's PreviousChecksum (so it doesn't match row 4's Checksum).
	entries[5].PreviousChecksum = "corrupted"

	repo := &recordingAuditRepo{entries: entries}
	svc := NewGovernanceService(repo)

	result, err := svc.VerifyChain(context.Background(), uuid.New(), now.Add(-time.Second), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected chain to be invalid")
	}
	if len(result.BrokenAt) != 1 {
		t.Errorf("expected 1 broken link, got %d", len(result.BrokenAt))
	}
	if result.BrokenAt[0].EntryID != 6 {
		t.Errorf("expected broken link at row 6 (id=6), got %d", result.BrokenAt[0].EntryID)
	}
}

func TestVerifyChain_EmptyRange(t *testing.T) {
	repo := &recordingAuditRepo{}
	svc := NewGovernanceService(repo)

	now := time.Now()
	result, err := svc.VerifyChain(context.Background(), uuid.New(), now.Add(-time.Hour), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Error("expected empty range to be valid")
	}
	if result.TotalRows != 0 {
		t.Errorf("expected 0 rows, got %d", result.TotalRows)
	}
}
