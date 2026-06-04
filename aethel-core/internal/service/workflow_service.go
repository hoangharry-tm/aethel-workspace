package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/google/uuid"

	"aethel-core/internal/domain"
)

// firstNoteAnchor is the sentinel previous_hash for the first note in a minute sheet.
const firstNoteAnchor = "genesis"

type WorkflowService struct {
	minuteSheets domain.MinuteSheetRepository
	greenNotes   domain.GreenNoteRepository
	audit        domain.AuditRepository
}

func NewWorkflowService(
	minuteSheets domain.MinuteSheetRepository,
	greenNotes domain.GreenNoteRepository,
	audit domain.AuditRepository,
) *WorkflowService {
	return &WorkflowService{
		minuteSheets: minuteSheets,
		greenNotes:   greenNotes,
		audit:        audit,
	}
}

func (s *WorkflowService) GetMinuteSheet(ctx context.Context, orgID, dispatchID uuid.UUID) (*domain.MinuteSheet, error) {
	return s.minuteSheets.GetByDispatchID(ctx, orgID, dispatchID)
}

func (s *WorkflowService) ListGreenNotes(ctx context.Context, orgID, minuteSheetID uuid.UUID) ([]domain.GreenNote, error) {
	return s.greenNotes.ListByMinuteSheet(ctx, orgID, minuteSheetID)
}

func (s *WorkflowService) AppendGreenNote(
	ctx context.Context,
	orgID, minuteSheetID, authorID uuid.UUID,
	content string,
	ip string,
) (*domain.GreenNote, error) {
	// Fetch the last note to determine the next sequence and validate the chain.
	lastNote, err := s.greenNotes.GetLastByMinuteSheet(ctx, orgID, minuteSheetID)
	if err != nil && err != domain.ErrNotFound {
		return nil, fmt.Errorf("fetch last green note: %w", err)
	}

	var prevHash string
	var nextSeq int

	if err == domain.ErrNotFound || lastNote == nil {
		prevHash = firstNoteAnchor
		nextSeq = 1
	} else {
		// Validate the chain integrity of the previous note before appending.
		expectedHash := computeNoteHash(lastNote.ContentBody, lastNote.SequenceOrder, lastNote.AuthorOfficerID, lastNote.PreviousHash)
		if expectedHash != lastNote.CryptographicHash {
			return nil, domain.ErrHashChainBroken
		}
		prevHash = lastNote.CryptographicHash
		nextSeq = lastNote.SequenceOrder + 1
	}

	hash := computeNoteHash(content, nextSeq, authorID, prevHash)

	note := &domain.GreenNote{
		ID:                uuid.New(),
		OrganizationID:    orgID,
		MinuteSheetID:     minuteSheetID,
		AuthorOfficerID:   authorID,
		SequenceOrder:     nextSeq,
		ContentBody:       content,
		CryptographicHash: hash,
		PreviousHash:      prevHash,
	}

	if err := s.greenNotes.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("create green note: %w", err)
	}

	_ = s.audit.Write(ctx, &domain.AuditEntry{
		OrganizationID:   orgID,
		ActorUserID:      &authorID,
		ActionEventType:  domain.AuditGreenNoteAppended,
		TargetResourceID: &minuteSheetID,
		IPAddress:        &ip,
	})

	return note, nil
}

func (s *WorkflowService) ApproveMinuteSheet(ctx context.Context, orgID, minuteSheetID, approverID uuid.UUID) error {
	if err := s.minuteSheets.Approve(ctx, orgID, minuteSheetID, approverID); err != nil {
		return err
	}
	_ = s.audit.Write(ctx, &domain.AuditEntry{
		OrganizationID:   orgID,
		ActorUserID:      &approverID,
		ActionEventType:  domain.AuditMinuteSheetApproved,
		TargetResourceID: &minuteSheetID,
	})
	return nil
}

// computeNoteHash computes SHA-256(content || sequence || authorID || prevHash).
func computeNoteHash(content string, seq int, authorID uuid.UUID, prevHash string) string {
	payload := content + strconv.Itoa(seq) + authorID.String() + prevHash
	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:])
}
