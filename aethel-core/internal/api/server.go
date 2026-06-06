// Package api wires together the HTTP router, middleware stack, and all
// request handlers for the Aethel backend.
package api

import (
	"aethel-core/internal/api/docs"
	"aethel-core/internal/api/handlers"
	"aethel-core/internal/app"
	"aethel-core/internal/config"
	"aethel-core/internal/database"
	"aethel-core/internal/domain"
	"aethel-core/internal/rbac"
	"aethel-core/internal/service"
	"aethel-core/internal/transport"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	apiMW "aethel-core/internal/api/middleware"
)

func init() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().Timestamp().Logger()
}

// Server holds all wired dependencies and the HTTP mux.
type Server struct {
	db          *sql.DB
	queries     *database.QueryRegistry
	configCache *config.ConfigCache
	sse         *transport.SSEBroker
	router      *chi.Mux
}

func NewServer(
	db *sql.DB,
	queries *database.QueryRegistry,
	configCache *config.ConfigCache,
	authSvc *handlers.AuthHandler,
	dispatchSvc *handlers.DispatchHandler,
	workflowSvc *handlers.WorkflowHandler,
	governanceSvc *service.GovernanceService,
	adminDeps handlers.AdminDeps,
) *Server {
	s := &Server{
		db:          db,
		queries:     queries,
		configCache: configCache,
		sse:         transport.NewSSEBroker(),
	}
	s.router = s.buildRouter(authSvc, dispatchSvc, workflowSvc, governanceSvc, adminDeps)
	return s
}

func (s *Server) ListenAndServe(addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Info().Str("addr", addr).Msg("HTTP server listening")
	return srv.ListenAndServe()
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) buildRouter(
	authSvc *handlers.AuthHandler,
	dispatchSvc *handlers.DispatchHandler,
	workflowSvc *handlers.WorkflowHandler,
	governanceSvc *service.GovernanceService,
	adminDeps handlers.AdminDeps,
) *chi.Mux {
	r := chi.NewRouter()

	// ── Global middleware stack (order matters) ───────────────────────────────
	// 1. Recovery — catch panics before any other middleware
	r.Use(middleware.Recoverer)
	// 2. Security headers — set on every response including panic recoveries
	r.Use(apiMW.SecurityHeaders)
	// 3. Request ID — for correlation in logs
	r.Use(middleware.RequestID)
	// 4. Structured request logger
	r.Use(zerologMiddleware)
	// 5. Global rate limit — 600 RPM per IP, covers unauthenticated traffic
	r.Use(apiMW.RateLimit)
	// 6. CORS
	r.Use(corsMiddleware)
	// 7. JWT extraction — sets user/role on context, injects app.OrgID
	r.Use(s.jwtMiddleware)
	// 8. Per-user rate limit — 300 RPM per authenticated user (post-JWT)
	r.Use(apiMW.RateLimitAuthenticated(func(req *http.Request) string {
		uid, _ := rbac.UserIDFromCtx(req.Context())
		return uid
	}))
	// 9. CSRF — rejects mutating requests where cookie ≠ header
	r.Use(apiMW.CSRFProtect)

	// ── Health probes ─────────────────────────────────────────────────────────
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := s.db.PingContext(r.Context()); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ready")
	})

	// ── API v1 ────────────────────────────────────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		// Config endpoints.
		cfgHandler := config.NewHandler(s.db, s.configCache)
		r.With(rbac.Require("dispatch.view")).Get("/config", cfgHandler.GetConfig)
		r.With(rbac.Require("dispatch.view")).Get("/config/branding", cfgHandler.GetBranding)
		r.With(rbac.Require("dispatch.view")).Get("/config/nav", cfgHandler.GetNav)
		r.With(rbac.Require("dispatch.view")).Get("/config/features", cfgHandler.GetFeatures)
		r.With(rbac.Require("admin.access")).Patch("/admin/config/branding", cfgHandler.PatchBranding)
		r.With(rbac.Require("admin.access")).Patch("/admin/config/nav", cfgHandler.PatchNav)
		r.With(rbac.Require("admin.access")).Patch("/admin/config/features", cfgHandler.PatchFeatures)
		r.With(rbac.Require("admin.access")).Patch("/admin/config/org", cfgHandler.PatchOrg)

		// Auth endpoints — login and refresh are public; logout requires auth.
		// Login has an additional strict per-IP rate limit (20 RPM).
		r.With(rbac.Require("public"), apiMW.RateLimitLogin).Post("/auth/login", authSvc.Login)
		r.With(rbac.Require("public")).Post("/auth/refresh", authSvc.Refresh)
		r.With(rbac.Require("dispatch.view")).Post("/auth/logout", authSvc.Logout)
		r.With(rbac.Require("public")).Post("/auth/password-reset/request", authSvc.RequestPasswordReset)
		r.With(rbac.Require("public")).Post("/auth/password-reset/confirm", authSvc.ConfirmPasswordReset)

		// Dispatch endpoints.
		r.With(rbac.Require("dispatch.view")).Get("/dispatches", dispatchSvc.ListInbox)
		r.With(rbac.Require("dispatch.create")).Post("/dispatches", dispatchSvc.Create)
		r.With(rbac.Require("dispatch.view")).Get("/dispatches/outbound", dispatchSvc.ListOutbound)
		r.With(rbac.Require("dispatch.create")).Post("/dispatches/outbound", dispatchSvc.CreateOutbound)
		r.With(rbac.Require("admin.access")).Get("/dispatches/unassigned", dispatchSvc.ListUnassigned)
		r.With(rbac.Require("workflow.view")).Get("/my-dispatches", dispatchSvc.ListMyDispatches)
		r.With(rbac.Require("dispatch.view")).Get("/search", dispatchSvc.Search)

		r.Route("/dispatches/{id}", func(r chi.Router) {
			r.With(rbac.Require("dispatch.view")).Get("/", dispatchSvc.GetByID)
			r.With(rbac.Require("dispatch.create")).Patch("/status", dispatchSvc.UpdateStatus)
			r.With(rbac.Require("dispatch.assign")).Post("/assign", dispatchSvc.Assign)
			r.With(rbac.Require("dispatch.deliver")).Post("/acknowledge", dispatchSvc.Acknowledge)
			r.With(rbac.Require("dispatch.view")).Get("/timeline", dispatchSvc.GetTimeline)
			r.With(rbac.Require("dispatch.view")).Get("/attachments", dispatchSvc.ListAttachments)
			r.With(rbac.Require("dispatch.create")).Post("/attachments", dispatchSvc.UploadAttachment)
			r.With(rbac.Require("dispatch.assign")).Delete("/attachments/{att_id}", dispatchSvc.DeleteAttachment)
			r.With(rbac.Require("workflow.view")).Get("/minute-sheet", workflowSvc.GetMinuteSheet)
			r.With(rbac.Require("workflow.view")).Get("/green-notes", workflowSvc.ListGreenNotes)
			r.With(rbac.Require("workflow.approve")).Post("/green-notes", workflowSvc.AppendGreenNote)
			r.With(rbac.Require("workflow.approve")).Post("/minute-sheet/approve", workflowSvc.ApproveMinuteSheet)
		})

		// Governance endpoints.
		gh := handlers.NewGovernanceHandler(governanceSvc)
		r.With(rbac.Require("admin.audit")).Get("/audit-log", gh.QueryAuditLog)
		r.With(rbac.Require("admin.audit")).Get("/audit-log/verify", gh.VerifyChain)

		// Admin endpoints.
		ah := handlers.NewAdminHandler(adminDeps)
		r.Route("/admin", func(r chi.Router) {
			r.Use(rbac.Require("admin.access"))
			r.Get("/users", ah.ListUsers)
			r.Post("/users", ah.CreateUser)
			r.Get("/users/{id}", ah.GetUser)
			r.Patch("/users/{id}", ah.UpdateUser)
			r.Delete("/users/{id}", ah.DeactivateUser)

			r.Get("/document-types", ah.ListDocumentTypes)
			r.Post("/document-types", ah.CreateDocumentType)
			r.Patch("/document-types/{id}", ah.UpdateDocumentType)
			r.Delete("/document-types/{id}", ah.DeleteDocumentType)

			r.Get("/routing-rules", ah.ListRoutingRules)
			r.Post("/routing-rules", ah.CreateRoutingRule)
			r.Put("/routing-rules/{id}", ah.UpdateRoutingRule)
			r.Delete("/routing-rules/{id}", ah.DeleteRoutingRule)

			r.Get("/escalation-rules", ah.ListEscalationRules)
			r.Post("/escalation-rules", ah.CreateEscalationRule)
			r.Put("/escalation-rules/{id}", ah.UpdateEscalationRule)

			r.Get("/reports", ah.GetReports)
			r.Get("/settings", ah.GetSettings)
			r.Patch("/settings", ah.UpdateSettings)
		})

		// Notifications (SSE) — Sprint 5.
		r.With(rbac.Require("dispatch.view")).Get("/notifications/stream", func(w http.ResponseWriter, r *http.Request) {
			userIDStr, _ := rbac.UserIDFromCtx(r.Context())
			s.sse.ServeHTTP(w, r, userIDStr)
		})
		r.With(rbac.Require("dispatch.view")).Get("/notifications", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintln(w, "[]")
		})
		r.With(rbac.Require("dispatch.view")).Patch("/notifications/{id}/read", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	})

	// API docs (Scalar UI + raw spec) — disabled by AETHEL_DISABLE_API_DOCS=true.
	// Register both patterns; handler uses full r.URL.Path to distinguish them.
	if docs.Enabled() {
		r.Handle("/api/docs", docs.Handler())
		r.Handle("/api/docs/*", docs.Handler())
	}

	return r
}

// ── Middleware ────────────────────────────────────────────────────────────────

func (s *Server) jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			next.ServeHTTP(w, r)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		secret := os.Getenv("AETHEL_JWT_SECRET")
		if secret == "" {
			secret = "dev-secret-change-in-production"
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			next.ServeHTTP(w, r)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		userID, _ := claims["sub"].(string)
		roleStr, _ := claims["role"].(string)
		role := domain.UserRole(roleStr)

		// Single-tenant: the JWT no longer carries an org claim.
		// Inject the boot-time OrgID so all handlers get a consistent value via rbac.OrgIDFromCtx.
		orgID := app.OrgID.String()

		ctx := rbac.SetUserContext(r.Context(), userID, orgID, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wildcard origin is incompatible with credentials:true — browsers reject it and
		// refuse to store Set-Cookie headers, breaking httpOnly cookie auth. Use explicit origin.
		origin := os.Getenv("AETHEL_CORS_ORIGIN")
		if origin == "" {
			origin = "http://localhost:3000"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func zerologMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", ww.Status()).
			Dur("latency", time.Since(start)).
			Str("request_id", middleware.GetReqID(r.Context())).
			Msg("request")
	})
}
