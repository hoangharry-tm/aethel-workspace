package middleware

import "net/http"

// MaxBodySize wraps the request body in an http.MaxBytesReader so that the
// handler receives at most limit bytes. Reading beyond the limit returns an
// error, which the framework (chi/stdlib) surfaces as a 413 or 400 response
// depending on where the overflow is detected.
//
// Place this middleware early in the stack — before the logger — so oversized
// payloads are rejected before any I/O-heavy processing begins.
func MaxBodySize(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
