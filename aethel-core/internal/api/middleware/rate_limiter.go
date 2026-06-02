package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// rateBucket is a per-key token bucket.
type rateBucket struct {
	mu       sync.Mutex
	tokens   float64
	lastFill time.Time
}

// rateLimiter holds per-key buckets with a shared capacity/rate config.
type rateLimiter struct {
	buckets  sync.Map // key string → *rateBucket
	rate     float64  // tokens refilled per second
	capacity float64  // max burst capacity (also the initial fill)
}

func newRateLimiter(perMinute int) *rateLimiter {
	cap := float64(perMinute)
	return &rateLimiter{
		rate:     cap / 60.0,
		capacity: cap,
	}
}

func (l *rateLimiter) allow(key string) bool {
	now := time.Now()
	v, _ := l.buckets.LoadOrStore(key, &rateBucket{tokens: l.capacity, lastFill: now})
	b := v.(*rateBucket)

	b.mu.Lock()
	defer b.mu.Unlock()

	elapsed := now.Sub(b.lastFill).Seconds()
	b.tokens = min(l.capacity, b.tokens+elapsed*l.rate)
	b.lastFill = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

var (
	globalLimiter = newRateLimiter(600) // 600 RPM per IP (pre-auth global)
	loginLimiter  = newRateLimiter(20)  // 20 RPM per IP (credential stuffing target)
	userLimiter   = newRateLimiter(300) // 300 RPM per authenticated user
)

func rateLimitError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
}

// RateLimit enforces global per-IP rate limiting (600 RPM).
// Placed before Auth middleware so it covers unauthenticated traffic.
func RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !globalLimiter.allow("ip:" + extractIP(r)) {
			rateLimitError(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RateLimitLogin applies the strict 20-RPM per-IP limit to the login endpoint.
// Applied as route-level middleware on POST /auth/login in addition to RateLimit.
func RateLimitLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !loginLimiter.allow("login:" + extractIP(r)) {
			rateLimitError(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RateLimitAuthenticated enforces 300 RPM per authenticated user ID.
// Reads the user ID already placed in context by the JWT middleware.
// Unauthenticated requests (no user ID) pass through — they are covered by RateLimit.
func RateLimitAuthenticated(userIDFromCtx func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if uid := userIDFromCtx(r); uid != "" {
				if !userLimiter.allow("user:" + uid) {
					rateLimitError(w)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
