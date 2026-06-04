package handlers

import (
	"aethel-core/internal/domain"
	"aethel-core/internal/rbac"
	"aethel-core/internal/service"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type DispatchHandler struct {
	svc *service.DispatchService
}

func NewDispatchHandler(svc *service.DispatchService) *DispatchHandler {
	return &DispatchHandler{svc: svc}
}

func (h *DispatchHandler) ListInbox(w http.ResponseWriter, r *http.Request) {
	orgID, deptID := mustOrgAndDept(r)
	page := pageFromQuery(r)

	dispatches, err := h.svc.ListInbox(r.Context(), orgID, deptID, page)
	if err != nil {
		writeError(w, "failed to load inbox", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, dispatches)
}

func (h *DispatchHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	userIDStr, _ := rbac.UserIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	userID, _ := uuid.Parse(userIDStr)

	var input service.Dispatch
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	d, err := h.svc.Create(r.Context(), &input, userID, orgID, clientIP(r))
	if err != nil {
		writeError(w, "failed to create dispatch", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (h *DispatchHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	orgID, id := mustOrgAndID(r)

	d, err := h.svc.GetByID(r.Context(), orgID, id)
	if err == domain.ErrNotFound {
		writeError(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		writeError(w, "failed to get dispatch", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *DispatchHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	orgID, id := mustOrgAndID(r)
	userIDStr, _ := rbac.UserIDFromCtx(r.Context())
	userID, _ := uuid.Parse(userIDStr)

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Status == "" {
		writeError(w, "status is required", http.StatusBadRequest)
		return
	}

	if err := h.svc.UpdateStatus(r.Context(), orgID, id, userID, domain.DispatchStatus(req.Status)); err != nil {
		writeError(w, "failed to update status", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DispatchHandler) Assign(w http.ResponseWriter, r *http.Request) {
	orgID, id := mustOrgAndID(r)
	userIDStr, _ := rbac.UserIDFromCtx(r.Context())
	actorID, _ := uuid.Parse(userIDStr)

	var req struct {
		UserID *string `json:"userId"`
		DeptID *string `json:"deptId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var userID, deptID *uuid.UUID
	if req.UserID != nil {
		parsed, err := uuid.Parse(*req.UserID)
		if err == nil {
			userID = &parsed
		}
	}
	if req.DeptID != nil {
		parsed, err := uuid.Parse(*req.DeptID)
		if err == nil {
			deptID = &parsed
		}
	}

	if err := h.svc.Assign(r.Context(), orgID, id, actorID, userID, deptID, clientIP(r)); err != nil {
		writeError(w, "failed to assign dispatch", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DispatchHandler) Acknowledge(w http.ResponseWriter, r *http.Request) {
	orgID, id := mustOrgAndID(r)
	userIDStr, _ := rbac.UserIDFromCtx(r.Context())
	userID, _ := uuid.Parse(userIDStr)

	if err := h.svc.Acknowledge(r.Context(), orgID, id, userID, clientIP(r)); err != nil {
		writeError(w, "failed to acknowledge dispatch", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DispatchHandler) ListOutbound(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	page := pageFromQuery(r)

	dispatches, err := h.svc.ListOutbound(r.Context(), orgID, page)
	if err != nil {
		writeError(w, "failed to list outbound", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, dispatches)
}

func (h *DispatchHandler) CreateOutbound(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	userIDStr, _ := rbac.UserIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	userID, _ := uuid.Parse(userIDStr)

	var input service.Dispatch
	input.Direction = "OUTBOUND"
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	input.Direction = "OUTBOUND"

	d, err := h.svc.Create(r.Context(), &input, userID, orgID, clientIP(r))
	if err != nil {
		writeError(w, "failed to create outbound dispatch", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (h *DispatchHandler) ListMyDispatches(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	userIDStr, _ := rbac.UserIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	userID, _ := uuid.Parse(userIDStr)
	page := pageFromQuery(r)

	dispatches, err := h.svc.ListByUser(r.Context(), orgID, userID, page)
	if err != nil {
		writeError(w, "failed to list dispatches", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, dispatches)
}

func (h *DispatchHandler) ListUnassigned(w http.ResponseWriter, r *http.Request) {
	page := pageFromQuery(r)
	dispatches, err := h.svc.ListUnassigned(r.Context(), page)
	if err != nil {
		writeError(w, "failed to list unassigned dispatches", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, dispatches)
}

func (h *DispatchHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	orgID, id := mustOrgAndID(r)

	events, err := h.svc.GetTimeline(r.Context(), orgID, id)
	if err != nil {
		writeError(w, "failed to get timeline", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// ListAttachments returns all attachments for a dispatch (stub — wired in Sprint 5).
func (h *DispatchHandler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

func (h *DispatchHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	writeError(w, "attachment storage not configured", http.StatusNotImplemented)
}

func (h *DispatchHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	writeError(w, "attachment storage not configured", http.StatusNotImplemented)
}

// Search placeholder — wired with full-text query in Sprint 5.
func (h *DispatchHandler) Search(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

func mustOrgAndID(r *http.Request) (orgID, id uuid.UUID) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ = uuid.Parse(orgIDStr)
	id, _ = uuid.Parse(chi.URLParam(r, "id"))
	return
}

func mustOrgAndDept(r *http.Request) (orgID, deptID uuid.UUID) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ = uuid.Parse(orgIDStr)
	return
}

func pageFromQuery(r *http.Request) domain.Page {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return domain.Page{Limit: limit, Offset: offset}
}
