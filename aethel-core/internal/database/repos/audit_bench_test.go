//go:build integration

package repos_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/google/uuid"

	"aethel-core/internal/blueprint"
	"aethel-core/internal/database"
	"aethel-core/internal/domain"
)

// BenchmarkAuditLedgerInsert benchmarks audit entry insert throughput.
// Skipped unless -tags integration is set and DATABASE_URL is configured.
func BenchmarkAuditLedgerInsert(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping DB benchmark in short mode")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		b.Skip("DATABASE_URL not set — skipping integration benchmark")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		b.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		b.Fatalf("ping db: %v", err)
	}

	// Load the query registry from the embedded queries.yaml.
	qCfg, err := blueprint.LoadQueriesConfig("../../database/queries/queries.yaml")
	if err != nil {
		b.Fatalf("load queries config: %v", err)
	}
	reg, err := database.BuildQueryRegistry(ctx, db, qCfg)
	if err != nil {
		b.Fatalf("build query registry: %v", err)
	}

	repo := NewAuditRepo(db, reg)

	orgID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()
	ip := "127.0.0.1"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		entry := &domain.AuditEntry{
			OrganizationID:  orgID,
			ActorUserID:     &actorID,
			ActionEventType: domain.AuditDispatchCreated,
			TargetResourceID: &targetID,
			IPAddress:       &ip,
			Metadata:        ptrStr(fmt.Sprintf(`{"bench_iter":%d}`, i)),
		}
		if err := repo.Write(context.Background(), entry); err != nil {
			b.Fatalf("Write audit entry: %v", err)
		}
	}
}

func ptrStr(s string) *string { return &s }
