package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"aethel-core/internal/domain"
	"aethel-core/internal/rbac"
)

// NotificationHandler implements the notification REST endpoints and SSE stream.
type NotificationHandler struct {
	repo      domain.NotificationRepository
	sseBroker interface {
		ServeHTTP(w http.ResponseWriter, r *http.Request, userID string)
	}
}

func NewNotificationHandler(deps NotificationDeps) *NotificationHandler {
	return &NotificationHandler{
		repo:      deps.Repo,
		sseBroker: deps.SSEBroker,
	}
}

// ListNotifications handles GET /api/v1/notifications
// Query params: ?page=1&per_page=20&unread_only=true
// Response: { "notifications": [...], "unread_count": N, "total": N }
func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	userIDStr, _ := rbac.UserIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	userID, _ := uuid.Parse(userIDStr)

	// Parse pagination.
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if perPage <= 0 || perPage > 100 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * perPage

	// Parse unread_only flag.
	unreadOnly := r.URL.Query().Get("unread_only") == "true"

	domainPage := domain.Page{Limit: perPage, Offset: offset}

	notifications, err := h.repo.ListByUser(r.Context(), orgID, userID, unreadOnly, domainPage)
	if err != nil {
		writeError(w, "failed to list notifications", http.StatusInternalServerError)
		return
	}

	unreadCount, err := h.repo.UnreadCount(r.Context(), orgID, userID)
	if err != nil {
		writeError(w, "failed to get unread count", http.StatusInternalServerError)
		return
	}

	// Ensure the slice is never null in JSON output.
	if notifications == nil {
		notifications = []domain.Notification{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": notifications,
		"unread_count":  unreadCount,
		"total":         len(notifications),
	})
}

// MarkNotificationRead handles PATCH /api/v1/notifications/{id}/read
// Marks a single notification as read. Returns 204 No Content.
func (h *NotificationHandler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	userIDStr, _ := rbac.UserIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	userID, _ := uuid.Parse(userIDStr)
	notifID, _ := uuid.Parse(chi.URLParam(r, "id"))

	if err := h.repo.MarkRead(r.Context(), orgID, notifID, userID); err == domain.ErrNotFound {
		writeError(w, "notification not found", http.StatusNotFound)
		return
	} else if err != nil {
		writeError(w, "failed to mark notification read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MarkAllNotificationsRead handles PATCH /api/v1/notifications/read-all
// Marks all notifications for the authenticated user as read. Returns 204 No Content.
func (h *NotificationHandler) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	orgIDStr, _ := rbac.OrgIDFromCtx(r.Context())
	userIDStr, _ := rbac.UserIDFromCtx(r.Context())
	orgID, _ := uuid.Parse(orgIDStr)
	userID, _ := uuid.Parse(userIDStr)

	if err := h.repo.MarkAllRead(r.Context(), orgID, userID); err != nil {
		writeError(w, "failed to mark all notifications read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// NotificationStream handles GET /api/v1/notifications/stream
// Upgrades the connection to Server-Sent Events (SSE).
// This endpoint is read-only and is exempt from CSRF by convention (GET).
func (h *NotificationHandler) NotificationStream(w http.ResponseWriter, r *http.Request) {
	userIDStr, _ := rbac.UserIDFromCtx(r.Context())
	h.sseBroker.ServeHTTP(w, r, userIDStr)
}
