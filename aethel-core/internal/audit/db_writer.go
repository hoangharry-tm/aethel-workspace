package audit

import (
	"context"

	"aethel-core/internal/domain"
)

// DBWriter implements Writer by delegating to an AuditRepository.
type DBWriter struct {
	repo domain.AuditRepository
}

// NewDBWriter returns a Writer backed by the given AuditRepository.
func NewDBWriter(repo domain.AuditRepository) *DBWriter {
	return &DBWriter{repo: repo}
}

func (w *DBWriter) Write(ctx context.Context, entry *domain.AuditEntry) error {
	return w.repo.Write(ctx, entry)
}
