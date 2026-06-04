package handlers

import (
	"aethel-core/internal/domain"
	"aethel-core/internal/rbac"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AdminHandler struct {
	users        domain.UserRepository
	docTypes     domain.DocumentTypeRepository
	routingRules domain.RoutingRuleRepository
	escRules     domain.EscalationRuleRepository
	audit        domain.AuditRepository
}

func NewAdminHandler(
	users domain.UserRepository,
	docTypes domain.DocumentTypeRepository,
	routingRules domain.RoutingRuleRepository,
	escRules domain.EscalationRuleRepository,
	audit domain.AuditRepository,
) *AdminHandler {
	return &AdminHandler{
		users:        users,
		docTypes:     docTypes,
		routingRules: routingRules,
		escRules:     escRules,
		audit:        audit,
	}
}

// ── Users ────────────────────────────────────────────────────────────────────

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)

	users, err := h.users.List(r.Context(), orgID, pageFromQuery(r))
	if err != nil {
		writeError(w, "failed to list users", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	actorIDStr, _ := rbac.UserIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	actorID, _ := uuid.Parse(actorIDStr)

	var u domain.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	u.ID = uuid.New()
	u.OrganizationID = orgID

	if err := h.users.Create(r.Context(), &u); err != nil {
		writeError(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	_ = h.audit.Write(r.Context(), &domain.AuditEntry{
		OrganizationID:   orgID,
		ActorUserID:      &actorID,
		ActionEventType:  domain.AuditAdminUserCreated,
		TargetResourceID: &u.ID,
	})

	writeJSON(w, http.StatusCreated, u)
}

func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	id, _ := uuid.Parse(chi.URLParam(r, "id"))

	u, err := h.users.GetByID(r.Context(), orgID, id)
	if err == domain.ErrNotFound {
		writeError(w, "user not found", http.StatusNotFound)
		return
	}
	if err != nil {
		writeError(w, "failed to get user", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	id, _ := uuid.Parse(chi.URLParam(r, "id"))

	u, err := h.users.GetByID(r.Context(), orgID, id)
	if err == domain.ErrNotFound {
		writeError(w, "user not found", http.StatusNotFound)
		return
	}
	if err != nil {
		writeError(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(u); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	u.ID = id
	u.OrganizationID = orgID

	if err := h.users.Update(r.Context(), u); err != nil {
		writeError(w, "failed to update user", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *AdminHandler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	actorIDStr, _ := rbac.UserIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	actorID, _ := uuid.Parse(actorIDStr)
	id, _ := uuid.Parse(chi.URLParam(r, "id"))

	u, err := h.users.GetByID(r.Context(), orgID, id)
	if err == domain.ErrNotFound {
		writeError(w, "user not found", http.StatusNotFound)
		return
	}
	if err != nil {
		writeError(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	u.IsActive = false
	if err := h.users.Update(r.Context(), u); err != nil {
		writeError(w, "failed to deactivate user", http.StatusInternalServerError)
		return
	}

	_ = h.audit.Write(r.Context(), &domain.AuditEntry{
		OrganizationID:   orgID,
		ActorUserID:      &actorID,
		ActionEventType:  domain.AuditAdminUserDeactivated,
		TargetResourceID: &id,
	})

	w.WriteHeader(http.StatusNoContent)
}

// ── Document Types ───────────────────────────────────────────────────────────

func (h *AdminHandler) ListDocumentTypes(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)

	types, err := h.docTypes.List(r.Context(), orgID)
	if err != nil {
		writeError(w, "failed to list document types", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, types)
}

func (h *AdminHandler) CreateDocumentType(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)

	var dt domain.DocumentType
	if err := json.NewDecoder(r.Body).Decode(&dt); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	dt.ID = uuid.New()
	dt.OrganizationID = orgID

	if err := h.docTypes.Create(r.Context(), &dt); err != nil {
		writeError(w, "failed to create document type", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, dt)
}

func (h *AdminHandler) UpdateDocumentType(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	id, _ := uuid.Parse(chi.URLParam(r, "id"))

	dt, err := h.docTypes.GetByID(r.Context(), orgID, id)
	if err == domain.ErrNotFound {
		writeError(w, "document type not found", http.StatusNotFound)
		return
	}
	if err != nil {
		writeError(w, "failed to get document type", http.StatusInternalServerError)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(dt); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	dt.ID = id
	dt.OrganizationID = orgID

	if err := h.docTypes.Update(r.Context(), dt); err != nil {
		writeError(w, "failed to update document type", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, dt)
}

func (h *AdminHandler) DeleteDocumentType(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	id, _ := uuid.Parse(chi.URLParam(r, "id"))

	if err := h.docTypes.Delete(r.Context(), orgID, id); err != nil {
		writeError(w, "failed to delete document type", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Routing Rules ────────────────────────────────────────────────────────────

func (h *AdminHandler) ListRoutingRules(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)

	rules, err := h.routingRules.List(r.Context(), orgID)
	if err != nil {
		writeError(w, "failed to list routing rules", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (h *AdminHandler) CreateRoutingRule(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	actorIDStr, _ := rbac.UserIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	actorID, _ := uuid.Parse(actorIDStr)

	var rule domain.RoutingRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	rule.ID = uuid.New()
	rule.OrganizationID = orgID

	if err := h.routingRules.Create(r.Context(), &rule); err != nil {
		writeError(w, "failed to create routing rule", http.StatusInternalServerError)
		return
	}

	_ = h.audit.Write(r.Context(), &domain.AuditEntry{
		OrganizationID:   orgID,
		ActorUserID:      &actorID,
		ActionEventType:  domain.AuditRoutingRuleModified,
		TargetResourceID: &rule.ID,
	})

	writeJSON(w, http.StatusCreated, rule)
}

func (h *AdminHandler) UpdateRoutingRule(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	id, _ := uuid.Parse(chi.URLParam(r, "id"))

	var rule domain.RoutingRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	rule.ID = id
	rule.OrganizationID = orgID

	if err := h.routingRules.Update(r.Context(), &rule); err != nil {
		writeError(w, "failed to update routing rule", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (h *AdminHandler) DeleteRoutingRule(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	id, _ := uuid.Parse(chi.URLParam(r, "id"))

	if err := h.routingRules.Delete(r.Context(), orgID, id); err != nil {
		writeError(w, "failed to delete routing rule", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Escalation Rules ─────────────────────────────────────────────────────────

func (h *AdminHandler) ListEscalationRules(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)

	rules, err := h.escRules.List(r.Context(), orgID)
	if err != nil {
		writeError(w, "failed to list escalation rules", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (h *AdminHandler) CreateEscalationRule(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)

	var rule domain.EscalationRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	rule.ID = uuid.New()
	rule.OrganizationID = orgID

	if err := h.escRules.Create(r.Context(), &rule); err != nil {
		writeError(w, "failed to create escalation rule", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h *AdminHandler) UpdateEscalationRule(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	id, _ := uuid.Parse(chi.URLParam(r, "id"))

	var rule domain.EscalationRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	rule.ID = id
	rule.OrganizationID = orgID

	if err := h.escRules.Update(r.Context(), &rule); err != nil {
		writeError(w, "failed to update escalation rule", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

// ── Reports / Settings ───────────────────────────────────────────────────────

func (h *AdminHandler) GetReports(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "reports endpoint — Sprint 5"})
}

func (h *AdminHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "settings endpoint — Sprint 5"})
}

func (h *AdminHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)
}
