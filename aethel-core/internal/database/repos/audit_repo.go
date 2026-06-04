package repos

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"aethel-core/internal/database"
	"aethel-core/internal/domain"
)

// AuditRepo implements domain.AuditRepository.
// The audit_ledger is INSERT-only — no UPDATE or DELETE is issued from this repo.
// After the db-harden.sql script is applied, the application DB user loses UPDATE/DELETE
// on audit_ledger at the PostgreSQL level as well.
type AuditRepo struct {
	db *sql.DB
	q  *database.QueryRegistry
}

func NewAuditRepo(db *sql.DB, q *database.QueryRegistry) *AuditRepo {
	return &AuditRepo{db: db, q: q}
}

// computeChecksum produces the SHA-256 chain checksum for an audit entry.
// The inputs are joined with ":" so that an empty previous checksum (first row) yields
// a distinct, deterministic hash.
func computeChecksum(eventType domain.AuditEventType, targetID *uuid.UUID, actorID *uuid.UUID, previousChecksum string) string {
	targetStr := ""
	if targetID != nil {
		targetStr = targetID.String()
	}
	actorStr := ""
	if actorID != nil {
		actorStr = actorID.String()
	}
	h := sha256.Sum256([]byte(strings.Join([]string{
		string(eventType), targetStr, actorStr, previousChecksum,
	}, ":")))
	return fmt.Sprintf("%x", h)
}

// Write appends an entry to the audit ledger.
// It first fetches the latest checksum to chain the entry, then inserts.
func (r *AuditRepo) Write(ctx context.Context, entry *domain.AuditEntry) error {
	// Fetch the previous (latest) checksum for this org so we can chain.
	var prevChecksum string
	row := r.q.Get("governance.get_latest_checksum").Stmt.QueryRowContext(ctx, entry.OrganizationID)
	if err := row.Scan(&prevChecksum); err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("audit_repo: get latest checksum: %w", err)
	}
	// If sql.ErrNoRows, prevChecksum stays "" — this is the first entry.

	checksum := computeChecksum(entry.ActionEventType, entry.TargetResourceID, entry.ActorUserID, prevChecksum)
	entry.PreviousChecksum = prevChecksum
	entry.Checksum = checksum

	// Map domain fields to migration column names:
	// domain.TargetTable   → target_resource_type
	// domain.IPAddress     → client_ip_address
	// domain.Metadata      → payload_snapshot
	// domain.Checksum      → payload_checksum
	_, err := r.q.Get("governance.write_audit_event").Stmt.ExecContext(ctx,
		entry.OrganizationID,  // $1: organization_id
		entry.ActorUserID,     // $2: actor_user_id (nullable)
		string(entry.ActionEventType), // $3: action_event_type
		entry.TargetResourceID, // $4: target_resource_id (nullable)
		entry.TargetTable,     // $5: target_resource_type (nullable)
		entry.IPAddress,       // $6: client_ip_address (nullable)
		entry.UserAgent,       // $7: user_agent (nullable)
		entry.Metadata,        // $8: payload_snapshot (nullable)
		entry.PreviousChecksum, // $9: previous_checksum
		entry.Checksum,        // $10: payload_checksum
	)
	return err
}

// Query returns audit entries in reverse chronological order for the given time window.
func (r *AuditRepo) Query(ctx context.Context, orgID uuid.UUID, from, to time.Time, page domain.Page) ([]domain.AuditEntry, error) {
	rows, err := r.q.Get("governance.query_audit_ledger_range").Stmt.QueryContext(ctx,
		orgID, from, to, page.Limit, page.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("audit_repo: query range: %w", err)
	}
	defer rows.Close()

	var result []domain.AuditEntry
	for rows.Next() {
		var e domain.AuditEntry
		var actorID uuid.NullUUID
		var targetID uuid.NullUUID
		var targetTable, ip, ua, metadata sql.NullString
		var prevChecksum sql.NullString

		if err := rows.Scan(
			&e.ID,
			&e.OrganizationID,
			&actorID,
			&e.ActionEventType,
			&targetID,
			&targetTable,  // target_resource_type → TargetTable
			&ip,           // client_ip_address → IPAddress
			&ua,           // user_agent → UserAgent
			&metadata,     // payload_snapshot → Metadata
			&prevChecksum, // previous_checksum
			&e.Checksum,   // payload_checksum → Checksum
			&e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("audit_repo: scan: %w", err)
		}

		if actorID.Valid {
			e.ActorUserID = &actorID.UUID
		}
		if targetID.Valid {
			e.TargetResourceID = &targetID.UUID
		}
		if targetTable.Valid {
			e.TargetTable = &targetTable.String
		}
		if ip.Valid {
			e.IPAddress = &ip.String
		}
		if ua.Valid {
			e.UserAgent = &ua.String
		}
		if metadata.Valid {
			e.Metadata = &metadata.String
		}
		if prevChecksum.Valid {
			e.PreviousChecksum = prevChecksum.String
		}

		result = append(result, e)
	}
	return result, rows.Err()
}

// VerifyChain re-computes the checksum for every entry in the time window (ascending order)
// and compares it against the stored payload_checksum. Any mismatch is recorded in BrokenAt.
func (r *AuditRepo) VerifyChain(ctx context.Context, orgID uuid.UUID, from, to time.Time) (*domain.ChainVerificationResult, error) {
	rows, err := r.q.Get("governance.verify_chain_range").Stmt.QueryContext(ctx, orgID, from, to)
	if err != nil {
		return nil, fmt.Errorf("audit_repo: verify_chain_range: %w", err)
	}
	defer rows.Close()

	var brokenAt []domain.BrokenLink
	count := 0

	for rows.Next() {
		count++
		var id int64
		var eventType domain.AuditEventType
		var targetID uuid.NullUUID
		var actorID uuid.NullUUID
		var prevChecksum sql.NullString
		var storedChecksum string
		var createdAt time.Time

		if err := rows.Scan(
			&id,
			&eventType,
			&targetID,
			&actorID,
			&prevChecksum,
			&storedChecksum, // payload_checksum
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("audit_repo: verify scan: %w", err)
		}

		var tID *uuid.UUID
		if targetID.Valid {
			tID = &targetID.UUID
		}
		var aID *uuid.UUID
		if actorID.Valid {
			aID = &actorID.UUID
		}
		prev := ""
		if prevChecksum.Valid {
			prev = prevChecksum.String
		}

		computed := computeChecksum(eventType, tID, aID, prev)
		if computed != storedChecksum {
			brokenAt = append(brokenAt, domain.BrokenLink{
				EntryID:          id,
				ComputedChecksum: computed,
				StoredChecksum:   storedChecksum,
				CreatedAt:        createdAt,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit_repo: verify rows: %w", err)
	}

	return &domain.ChainVerificationResult{
		Valid:     len(brokenAt) == 0,
		TotalRows: count,
		BrokenAt:  brokenAt,
	}, nil
}
