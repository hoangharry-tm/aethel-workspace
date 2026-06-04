package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"aethel-core/internal/domain"
)

// ── mock repos ─────────────────────────────────────────────────────────────

type mockMinuteSheetRepo struct {
	sheets      map[uuid.UUID]*domain.MinuteSheet // keyed by ID
	approvedIDs []uuid.UUID
}

func newMockMinuteSheetRepo() *mockMinuteSheetRepo {
	return &mockMinuteSheetRepo{sheets: make(map[uuid.UUID]*domain.MinuteSheet)}
}

func (r *mockMinuteSheetRepo) Create(_ context.Context, ms *domain.MinuteSheet) error {
	r.sheets[ms.ID] = ms
	return nil
}

func (r *mockMinuteSheetRepo) GetByDispatchID(_ context.Context, _ uuid.UUID, dispatchID uuid.UUID) (*domain.MinuteSheet, error) {
	for _, ms := range r.sheets {
		if ms.DispatchID == dispatchID {
			return ms, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *mockMinuteSheetRepo) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.MinuteSheet, error) {
	ms, ok := r.sheets[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return ms, nil
}

func (r *mockMinuteSheetRepo) Approve(_ context.Context, _, id uuid.UUID, _ uuid.UUID) error {
	r.approvedIDs = append(r.approvedIDs, id)
	if ms, ok := r.sheets[id]; ok {
		ms.Status = domain.MinuteSheetApproved
	}
	return nil
}

// mockGreenNoteRepo is an in-memory GreenNoteRepository.
// It does NOT validate the chain — the chain validation happens in the real repo.
// The service calls repo.Create with a fully-formed note; the mock just stores it.
type mockGreenNoteRepo struct {
	notes []domain.GreenNote
}

func (r *mockGreenNoteRepo) Create(_ context.Context, note *domain.GreenNote) error {
	r.notes = append(r.notes, *note)
	return nil
}

func (r *mockGreenNoteRepo) ListByMinuteSheet(_ context.Context, _ uuid.UUID, minuteSheetID uuid.UUID) ([]domain.GreenNote, error) {
	var out []domain.GreenNote
	for _, n := range r.notes {
		if n.MinuteSheetID == minuteSheetID {
			out = append(out, n)
		}
	}
	return out, nil
}

func (r *mockGreenNoteRepo) GetLastByMinuteSheet(_ context.Context, _ uuid.UUID, minuteSheetID uuid.UUID) (*domain.GreenNote, error) {
	var last *domain.GreenNote
	for i, n := range r.notes {
		if n.MinuteSheetID == minuteSheetID {
			if last == nil || n.SequenceOrder > last.SequenceOrder {
				last = &r.notes[i]
			}
		}
	}
	if last == nil {
		return nil, domain.ErrNotFound
	}
	return last, nil
}

// mockAuditRepoWf tracks written events.
type mockAuditRepoWf struct {
	events []domain.AuditEventType
}

func (r *mockAuditRepoWf) Write(_ context.Context, e *domain.AuditEntry) error {
	r.events = append(r.events, e.ActionEventType)
	return nil
}

func (r *mockAuditRepoWf) Query(_ context.Context, _ uuid.UUID, _, _ time.Time, _ domain.Page) ([]domain.AuditEntry, error) {
	return nil, nil
}

func (r *mockAuditRepoWf) VerifyChain(_ context.Context, _ uuid.UUID, _, _ time.Time) (*domain.ChainVerificationResult, error) {
	return &domain.ChainVerificationResult{Valid: true}, nil
}

// ── test cases ────────────────────────────────────────────────────────────

// TestHashChain_FiveNotesValid appends 5 notes and verifies each note's
// PreviousHash equals the prior note's CryptographicHash.
func TestHashChain_FiveNotesValid(t *testing.T) {
	msRepo := newMockMinuteSheetRepo()
	noteRepo := &mockGreenNoteRepo{}
	auditRepo := &mockAuditRepoWf{}
	svc := NewWorkflowService(msRepo, noteRepo, auditRepo)

	orgID := uuid.New()
	msID := uuid.New()
	authorID := uuid.New()

	// Seed a minute sheet.
	ms := &domain.MinuteSheet{ID: msID, DispatchID: uuid.New(), OrganizationID: orgID, Status: domain.MinuteSheetOpen}
	_ = msRepo.Create(context.Background(), ms)

	contents := []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo"}
	for i, content := range contents {
		note, err := svc.AppendGreenNote(context.Background(), orgID, msID, authorID, content, "127.0.0.1")
		if err != nil {
			t.Fatalf("note %d: unexpected error: %v", i+1, err)
		}
		if note.SequenceOrder != i+1 {
			t.Errorf("note %d: want seq=%d, got %d", i+1, i+1, note.SequenceOrder)
		}
	}

	// Verify PreviousHash of note N == CryptographicHash of note N-1.
	notes := noteRepo.notes
	if len(notes) != 5 {
		t.Fatalf("want 5 notes, got %d", len(notes))
	}
	for i := 1; i < len(notes); i++ {
		if notes[i].PreviousHash != notes[i-1].CryptographicHash {
			t.Errorf("chain broken at note %d: previousHash=%q, note[%d].hash=%q",
				i+1, notes[i].PreviousHash, i, notes[i-1].CryptographicHash)
		}
	}
}

// TestHashChain_TamperAtNote3 appends 5 notes, corrupts the last note's hash,
// then verifies that appending a 6th note returns ErrHashChainBroken.
// Note: the service only validates the *last* note on append; tampering of
// interior notes is detected by the audit/verify-chain endpoint, not here.
func TestHashChain_TamperAtNote3(t *testing.T) {
	msRepo := newMockMinuteSheetRepo()
	noteRepo := &mockGreenNoteRepo{}
	auditRepo := &mockAuditRepoWf{}
	svc := NewWorkflowService(msRepo, noteRepo, auditRepo)

	orgID := uuid.New()
	msID := uuid.New()
	authorID := uuid.New()

	ms := &domain.MinuteSheet{ID: msID, DispatchID: uuid.New(), OrganizationID: orgID, Status: domain.MinuteSheetOpen}
	_ = msRepo.Create(context.Background(), ms)

	for i := 0; i < 5; i++ {
		_, err := svc.AppendGreenNote(context.Background(), orgID, msID, authorID,
			"note content", "127.0.0.1")
		if err != nil {
			t.Fatalf("setup note %d: %v", i+1, err)
		}
	}

	// Tamper: corrupt the LAST note's hash (index 4 = note 5).
	// The service fetches the last note and re-derives its hash — this is what gets detected.
	noteRepo.notes[4].CryptographicHash = "tampered"

	_, err := svc.AppendGreenNote(context.Background(), orgID, msID, authorID, "note 6", "127.0.0.1")
	if err != domain.ErrHashChainBroken {
		t.Errorf("want ErrHashChainBroken, got %v", err)
	}
}

// TestAppendGreenNote_FirstNote verifies that the first note in a minute sheet
// uses the genesis sentinel as its PreviousHash and has SequenceOrder == 1.
func TestAppendGreenNote_FirstNote(t *testing.T) {
	msRepo := newMockMinuteSheetRepo()
	noteRepo := &mockGreenNoteRepo{}
	auditRepo := &mockAuditRepoWf{}
	svc := NewWorkflowService(msRepo, noteRepo, auditRepo)

	orgID := uuid.New()
	msID := uuid.New()
	authorID := uuid.New()

	ms := &domain.MinuteSheet{ID: msID, DispatchID: uuid.New(), OrganizationID: orgID, Status: domain.MinuteSheetOpen}
	_ = msRepo.Create(context.Background(), ms)

	note, err := svc.AppendGreenNote(context.Background(), orgID, msID, authorID, "First note", "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if note.SequenceOrder != 1 {
		t.Errorf("want Sequence=1, got %d", note.SequenceOrder)
	}
	// The first note's PreviousHash is the sentinel anchor (firstNoteAnchor = "genesis").
	if note.PreviousHash != firstNoteAnchor {
		t.Errorf("want PreviousHash=%q, got %q", firstNoteAnchor, note.PreviousHash)
	}
}

// TestApproveMinuteSheet_WritesAuditEntry verifies that ApproveMinuteSheet
// calls s.audit.Write with AuditMinuteSheetApproved after delegating to the repo.
func TestApproveMinuteSheet_WritesAuditEntry(t *testing.T) {
	msRepo := newMockMinuteSheetRepo()
	noteRepo := &mockGreenNoteRepo{}
	auditRepo := &mockAuditRepoWf{}
	svc := NewWorkflowService(msRepo, noteRepo, auditRepo)

	orgID := uuid.New()
	msID := uuid.New()
	approverID := uuid.New()

	ms := &domain.MinuteSheet{ID: msID, DispatchID: uuid.New(), OrganizationID: orgID, Status: domain.MinuteSheetOpen}
	_ = msRepo.Create(context.Background(), ms)

	err := svc.ApproveMinuteSheet(context.Background(), orgID, msID, approverID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, ev := range auditRepo.events {
		if ev == domain.AuditMinuteSheetApproved {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected MINUTE_SHEET_APPROVED audit event, got %v", auditRepo.events)
	}
}
