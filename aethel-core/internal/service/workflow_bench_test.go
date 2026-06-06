package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"aethel-core/internal/domain"
)

// BenchmarkHashChainBuild benchmarks building a 100-note chain by calling
// AppendGreenNote 100 times on the same minute sheet.
// Uses the same mock types already defined in workflow_service_test.go.
func BenchmarkHashChainBuild(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		msRepo := newMockMinuteSheetRepo()
		noteRepo := &mockGreenNoteRepo{}
		auditRepo := &mockAuditRepoWf{}
		svc := NewWorkflowService(msRepo, noteRepo, auditRepo)

		orgID := uuid.New()
		msID := uuid.New()
		authorID := uuid.New()

		ms := &domain.MinuteSheet{
			ID:             msID,
			DispatchID:     uuid.New(),
			OrganizationID: orgID,
			Status:         domain.MinuteSheetOpen,
		}
		_ = msRepo.Create(context.Background(), ms)

		b.StartTimer()

		for j := 0; j < 100; j++ {
			content := fmt.Sprintf("note content %d", j)
			_, err := svc.AppendGreenNote(context.Background(), orgID, msID, authorID, content, "127.0.0.1")
			if err != nil {
				b.Fatalf("AppendGreenNote iteration %d: %v", j, err)
			}
		}
	}
}

// BenchmarkHashChainVerify benchmarks verifying an already-built 100-note chain
// by re-deriving each note's hash from its stored fields and comparing to the stored value.
// This mirrors the logic in AuditRepo.VerifyChain but for green notes in-memory.
func BenchmarkHashChainVerify(b *testing.B) {
	// Build the chain once outside the timer.
	msRepo := newMockMinuteSheetRepo()
	noteRepo := &mockGreenNoteRepo{}
	auditRepo := &mockAuditRepoWf{}
	svc := NewWorkflowService(msRepo, noteRepo, auditRepo)

	orgID := uuid.New()
	msID := uuid.New()
	authorID := uuid.New()

	ms := &domain.MinuteSheet{
		ID:             msID,
		DispatchID:     uuid.New(),
		OrganizationID: orgID,
		Status:         domain.MinuteSheetOpen,
	}
	_ = msRepo.Create(context.Background(), ms)

	for j := 0; j < 100; j++ {
		_, err := svc.AppendGreenNote(context.Background(), orgID, msID, authorID,
			fmt.Sprintf("bench note %d", j), "127.0.0.1")
		if err != nil {
			b.Fatalf("setup AppendGreenNote %d: %v", j, err)
		}
	}

	notes := noteRepo.notes // snapshot the built chain

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Re-derive each note's hash and compare to stored value (full chain walk).
		for k, note := range notes {
			computed := computeNoteHash(note.ContentBody, note.SequenceOrder, note.AuthorOfficerID, note.PreviousHash)
			if computed != note.CryptographicHash {
				b.Fatalf("chain broken at note %d (index %d)", note.SequenceOrder, k)
			}
		}
	}
}
