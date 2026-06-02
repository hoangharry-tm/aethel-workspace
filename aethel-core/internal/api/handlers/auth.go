package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"aethel-core/internal/app"
	"aethel-core/internal/domain"
	"aethel-core/internal/rbac"
	"aethel-core/internal/service"
)

const refreshTokenMaxAge = int(7 * 24 * time.Hour / time.Second)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Role        string `json:"role"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, "email and password are required", http.StatusBadRequest)
		return
	}

	result, err := h.svc.Login(r.Context(), app.OrgID, req.Email, req.Password, clientIP(r), r.UserAgent())
	if err != nil {
		switch err {
		case domain.ErrUnauthorized:
			writeError(w, "invalid credentials", http.StatusUnauthorized)
		case domain.ErrAccountLocked:
			writeError(w, "account temporarily locked due to too many failed attempts", http.StatusLocked)
		default:
			writeError(w, "login failed", http.StatusInternalServerError)
		}
		return
	}

	csrfToken := generateCSRFToken()
	isHTTPS := r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil

	// Refresh token: httpOnly — JavaScript cannot read it.
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    result.RefreshToken,
		Path:     "/api/v1/auth/refresh",
		HttpOnly: true,
		Secure:   isHTTPS,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   refreshTokenMaxAge,
	})

	// CSRF token: NOT httpOnly — JavaScript reads it and sends as X-CSRF-Token header.
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: false,
		Secure:   isHTTPS,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   refreshTokenMaxAge,
	})

	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(15 * 60), // 15 minutes in seconds
		Role:        string(result.User.Role),
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		writeError(w, "refresh token cookie missing", http.StatusUnauthorized)
		return
	}

	result, err := h.svc.RefreshSession(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, "invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	csrfToken := generateCSRFToken()
	isHTTPS := r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    result.RefreshToken,
		Path:     "/api/v1/auth/refresh",
		HttpOnly: true,
		Secure:   isHTTPS,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   refreshTokenMaxAge,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: false,
		Secure:   isHTTPS,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   refreshTokenMaxAge,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": result.AccessToken,
		"expires_in":   int(15 * 60),
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := rbac.UserIDFromCtx(r.Context())
	if !ok {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, _ := uuid.Parse(userIDStr)

	var rawRefreshToken string
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		rawRefreshToken = cookie.Value
	}

	_ = h.svc.Logout(r.Context(), app.OrgID, userID, rawRefreshToken, clientIP(r), r.UserAgent())

	// Clear cookies by setting MaxAge -1.
	clearCookie(w, "refresh_token", "/api/v1/auth/refresh")
	clearCookie(w, "csrf_token", "/")

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Always return 202 — do not reveal whether the email exists.
	_ = h.svc.RequestPasswordReset(r.Context(), app.OrgID, req.Email)
	w.WriteHeader(http.StatusAccepted)
}

func (h *AuthHandler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		writeError(w, "token and newPassword are required", http.StatusBadRequest)
		return
	}

	if err := h.svc.ConfirmPasswordReset(r.Context(), req.Token, req.NewPassword); err != nil {
		writeError(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// generateCSRFToken returns a cryptographically random base64url string (32 bytes).
func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// This should never happen on a working OS. Fail loudly rather than silently.
		panic("generateCSRFToken: crypto/rand.Read failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func clearCookie(w http.ResponseWriter, name, path string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		HttpOnly: true,
		MaxAge:   -1,
	})
}
