package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"aethel-core/internal/rbac"
	"aethel-core/internal/service"
)

type GovernanceHandler struct {
	svc *service.GovernanceService
}

func NewGovernanceHandler(svc *service.GovernanceService) *GovernanceHandler {
	return &GovernanceHandler{svc: svc}
}

func (h *GovernanceHandler) QueryAuditLog(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)

	from, to := parseTimeRange(r)
	page := pageFromQuery(r)

	entries, err := h.svc.QueryAuditLog(r.Context(), orgID, from, to, page)
	if err != nil {
		writeError(w, "failed to query audit log", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *GovernanceHandler) VerifyChain(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)

	from, to := parseTimeRange(r)

	result, err := h.svc.VerifyChain(r.Context(), orgID, from, to)
	if err != nil {
		writeError(w, "failed to verify chain", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parseTimeRange(r *http.Request) (from, to time.Time) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from, _ = time.Parse(time.RFC3339, fromStr)
	to, _ = time.Parse(time.RFC3339, toStr)

	if from.IsZero() {
		from = time.Now().AddDate(0, -1, 0)
	}
	if to.IsZero() {
		to = time.Now()
	}
	return
}
