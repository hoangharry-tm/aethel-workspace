package middleware

import "net/http"

// SecurityHeaders sets security-relevant response headers on every response.
// Must be first after Recovery so headers are present even on panic recovery responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		// Disabled intentionally: modern browsers rely on CSP; this header causes bugs in legacy IE.
		h.Set("X-XSS-Protection", "0")
		// HSTS only over HTTPS — check the reverse-proxy forwarding header.
		if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		// API responses must not be cached by intermediaries.
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
