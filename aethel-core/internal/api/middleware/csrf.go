package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

// CSRFProtect implements the double-submit cookie pattern for browser clients.
//
// Algorithm:
//  1. Read-only methods (GET, HEAD, OPTIONS): pass through.
//  2. If no "csrf_token" cookie: pass through (non-browser / API client path).
//  3. Read the X-CSRF-Token request header.
//  4. Compare with crypto/subtle.ConstantTimeCompare to prevent timing attacks.
//  5. Mismatch → 403 {"error":"csrf token mismatch"}.
//
// The csrf_token cookie is set by the login handler as a readable (non-httpOnly) cookie.
// JavaScript reads it and sends it back as the X-CSRF-Token header on every mutating request.
func CSRFProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("csrf_token")
		if err != nil {
			// No CSRF cookie — treat as a non-browser API client and pass through.
			next.ServeHTTP(w, r)
			return
		}

		header := r.Header.Get("X-CSRF-Token")
		if subtle.ConstantTimeCompare([]byte(header), []byte(cookie.Value)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "csrf token mismatch"})
			return
		}

		next.ServeHTTP(w, r)
	})
}
