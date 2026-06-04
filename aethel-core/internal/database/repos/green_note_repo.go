package repos

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strconv"

	"github.com/google/uuid"

	"aethel-core/internal/database"
	"aethel-core/internal/domain"
)

// GreenNoteRepo implements domain.GreenNoteRepository.
// green_notes rows are INSERT-only — no UPDATE or DELETE ever.
// Single-tenant: the green_notes table has no organization_id column in this schema version.
type GreenNoteRepo struct {
	db *sql.DB
	q  *database.QueryRegistry
}

func NewGreenNoteRepo(db *sql.DB, q *database.QueryRegistry) *GreenNoteRepo {
	return &GreenNoteRepo{db: db, q: q}
}

// Create opens a transaction, locks the last note row with SELECT FOR UPDATE,
// validates the hash chain, then inserts the new note — all atomically.
// This prevents concurrent appends from computing duplicate sequence numbers.
func (r *GreenNoteRepo) Create(ctx context.Context, note *domain.GreenNote) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	// Lock the tail of the chain while we inspect it.
	last, err := fetchLastInTx(ctx, tx, r.q, note.MinuteSheetID)
	if err != nil && err != domain.ErrNotFound {
		return err
	}

	if last != nil {
		// Validate: re-derive last note's hash from its stored fields.
		// Must use the same formula as workflow_service.go computeNoteHash.
		expected := computeGreenNoteHash(last.ContentBody, last.SequenceOrder, last.AuthorOfficerID, last.PreviousHash)
		if expected != last.CryptographicHash {
			return domain.ErrHashChainBroken
		}
	}

	// Insert the new note within the same transaction.
	stmt := tx.StmtContext(ctx, r.q.Get("green_note.insert").Stmt)
	row := stmt.QueryRowContext(ctx,
		note.ID,
		note.MinuteSheetID,
		note.SequenceOrder,
		note.AuthorOfficerID,
		note.ContentBody,
		note.CryptographicHash,
		note.PreviousHash,
		note.DigitalSignature != nil,
		note.DigitalSignature,
	)
	if err := scanGreenNote(row, note); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *GreenNoteRepo) ListByMinuteSheet(ctx context.Context, _ uuid.UUID, minuteSheetID uuid.UUID) ([]domain.GreenNote, error) {
	rows, err := r.q.Get("green_note.fetch_all").Stmt.QueryContext(ctx, minuteSheetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGreenNotes(rows)
}

func (r *GreenNoteRepo) GetLastByMinuteSheet(ctx context.Context, _ uuid.UUID, minuteSheetID uuid.UUID) (*domain.GreenNote, error) {
	// Non-transactional read — used by service layer for display, not for append.
	// For append safety, fetchLastInTx is used inside Create.
	row := r.q.Get("green_note.fetch_last_for_update").Stmt.QueryRowContext(ctx, minuteSheetID)
	note := &domain.GreenNote{}
	if err := scanGreenNote(row, note); err != nil {
		return nil, err
	}
	return note, nil
}

// fetchLastInTx fetches the last green note for a minute sheet within a transaction,
// locking the row (FOR UPDATE) to prevent concurrent appends.
func fetchLastInTx(ctx context.Context, tx *sql.Tx, q *database.QueryRegistry, minuteSheetID uuid.UUID) (*domain.GreenNote, error) {
	stmt := tx.StmtContext(ctx, q.Get("green_note.fetch_last_for_update").Stmt)
	row := stmt.QueryRowContext(ctx, minuteSheetID)
	note := &domain.GreenNote{}
	if err := scanGreenNote(row, note); err != nil {
		return nil, err
	}
	return note, nil
}

func scanGreenNote(row *sql.Row, n *domain.GreenNote) error {
	var sig *string
	err := row.Scan(
		&n.ID,
		&n.MinuteSheetID,
		&n.SequenceOrder,
		&n.AuthorOfficerID,
		&n.ContentBody,
		&n.CryptographicHash,
		&n.PreviousHash,
		new(bool), // is_signed — not in domain struct; discard
		&sig,
		&n.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	n.DigitalSignature = sig
	return nil
}

func scanGreenNotes(rows *sql.Rows) ([]domain.GreenNote, error) {
	var result []domain.GreenNote
	for rows.Next() {
		n := domain.GreenNote{}
		var sig *string
		err := rows.Scan(
			&n.ID,
			&n.MinuteSheetID,
			&n.SequenceOrder,
			&n.AuthorOfficerID,
			&n.ContentBody,
			&n.CryptographicHash,
			&n.PreviousHash,
			new(bool),
			&sig,
			&n.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		n.DigitalSignature = sig
		result = append(result, n)
	}
	return result, rows.Err()
}

// computeGreenNoteHash computes SHA-256(content || sequence || authorID || prevHash).
// IMPORTANT: This formula MUST match computeNoteHash in workflow_service.go exactly.
// The canonical formula concatenates without separators.
func computeGreenNoteHash(content string, seq int, authorID uuid.UUID, prevHash string) string {
	payload := content + strconv.Itoa(seq) + authorID.String() + prevHash
	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:])
}
