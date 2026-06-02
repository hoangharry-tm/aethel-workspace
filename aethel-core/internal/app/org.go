package app

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// OrgID is the UUID of the single organization row for this installation.
// Set once at startup by LoadOrgID; read everywhere else.
// Never call LoadOrgID after startup.
var OrgID uuid.UUID

// LoadOrgID queries SELECT id FROM organizations LIMIT 1 and stores the result.
// Returns an error if the table is empty — the caller (main.go) should handle
// this by prompting the operator to run the setup script.
func LoadOrgID(ctx context.Context, db *sql.DB) error {
	row := db.QueryRowContext(ctx, `SELECT id FROM organizations LIMIT 1`)
	var id uuid.UUID
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no organization found: run the setup script to initialize the installation")
		}
		return fmt.Errorf("load org ID: %w", err)
	}
	OrgID = id
	return nil
}
