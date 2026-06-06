package middleware

import (
	"net/http"
	"time"

	"aethel-core/internal/rbac"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

// responseWriter wraps http.ResponseWriter to capture the status code written by the handler.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// StructuredLogger logs every request with zerolog fields:
//
//	request_id, method, path, status_code, latency_ms, user_id (if present), remote_ip
//
// Log level: INFO for 2xx/3xx, WARN for 4xx, ERROR for 5xx.
// Health probe paths (/health, /ready) are skipped to reduce noise.
func StructuredLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip liveness / readiness probes.
			if r.URL.Path == "/health" || r.URL.Path == "/ready" || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			ww := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(ww, r)

			latencyMs := time.Since(start).Milliseconds()
			status := ww.status

			ev := logger.Info()
			switch {
			case status >= 500:
				ev = logger.Error()
			case status >= 400:
				ev = logger.Warn()
			}

			// Correlation ID set by chi's RequestID middleware.
			reqID := chimw.GetReqID(r.Context())

			// User ID — present only when the JWT middleware has run.
			userID, _ := rbac.UserIDFromCtx(r.Context())

			event := ev.
				Str("request_id", reqID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status_code", status).
				Int64("latency_ms", latencyMs).
				Str("remote_ip", extractIP(r))

			if userID != "" {
				event = event.Str("user_id", userID)
			}

			event.Msg("request")
		})
	}
}
