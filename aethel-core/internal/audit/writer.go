package audit

import (
	"context"

	"aethel-core/internal/domain"
)

// Writer is the single audit write path for the entire application.
// Services that record audit events receive a Writer via constructor injection —
// never call AuditRepository directly from services.
type Writer interface {
	Write(ctx context.Context, entry *domain.AuditEntry) error
}
