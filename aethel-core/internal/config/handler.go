package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"aethel-core/internal/app"
)

// Handler exposes GET /api/v1/config and PATCH /api/v1/admin/config/* endpoints.
type Handler struct {
	db    *sql.DB
	cache *ConfigCache
}

func NewHandler(db *sql.DB, cache *ConfigCache) *Handler {
	return &Handler{db: db, cache: cache}
}

type ctxKey string

// OrgIDContextKey is kept exported so the tenant middleware in api/server.go can compile.
// The config handlers do not read from this key — they use app.OrgID directly.
const OrgIDContextKey ctxKey = "orgID"

// ── GET endpoints (cache-first) ───────────────────────────────────────────────

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.loadCached(r.Context())
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	writeJSON(w, cfg)
}

func (h *Handler) GetBranding(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.loadCached(r.Context())
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	writeJSON(w, cfg.Branding)
}

func (h *Handler) GetNav(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.loadCached(r.Context())
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	writeJSON(w, cfg.Nav)
}

func (h *Handler) GetFeatures(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.loadCached(r.Context())
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	writeJSON(w, cfg.Features)
}

// ── PATCH endpoints (write to DB, invalidate cache) ──────────────────────────

func (h *Handler) PatchBranding(w http.ResponseWriter, r *http.Request) {
	var input BrandingConfig
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONErr(w, "invalid request body", http.StatusBadRequest)
		return
	}

	orgID := app.OrgID
	_, err := h.db.ExecContext(r.Context(), `
		INSERT INTO branding_configs (id, organization_id, primary_brand_color, neutral_palette, font_family, wordmark, logo_file_path, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, now())
		ON CONFLICT (organization_id) DO UPDATE SET
			primary_brand_color = COALESCE(NULLIF(EXCLUDED.primary_brand_color, ''), branding_configs.primary_brand_color),
			neutral_palette     = COALESCE(NULLIF(EXCLUDED.neutral_palette, ''),     branding_configs.neutral_palette),
			font_family         = COALESCE(NULLIF(EXCLUDED.font_family, ''),         branding_configs.font_family),
			wordmark            = COALESCE(NULLIF(EXCLUDED.wordmark, ''),             branding_configs.wordmark),
			logo_file_path      = COALESCE(NULLIF(EXCLUDED.logo_file_path, ''),      branding_configs.logo_file_path),
			updated_at          = now()
	`, orgID,
		nullIfEmpty(input.PrimaryColor),
		nullIfEmpty(input.NeutralPalette),
		nullIfEmpty(input.FontFamily),
		nullIfEmpty(input.Wordmark),
		nullIfEmpty(input.LogoPath),
	)
	if err != nil {
		writeJSONErr(w, "failed to update branding", http.StatusInternalServerError)
		return
	}

	h.cache.Invalidate()
	cfg, err := h.loadCached(r.Context())
	if err != nil {
		writeJSONErr(w, "failed to reload config", http.StatusInternalServerError)
		return
	}
	writeJSON(w, cfg.Branding)
}

func (h *Handler) PatchNav(w http.ResponseWriter, r *http.Request) {
	var nav []NavGroup
	if err := json.NewDecoder(r.Body).Decode(&nav); err != nil {
		writeJSONErr(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(nav) == 0 {
		writeJSONErr(w, "nav must be a non-empty array", http.StatusBadRequest)
		return
	}

	navJSON, err := json.Marshal(nav)
	if err != nil {
		writeJSONErr(w, "failed to encode nav", http.StatusInternalServerError)
		return
	}

	orgID := app.OrgID
	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO system_settings (id, organization_id, key, value, value_type, updated_at)
		VALUES (gen_random_uuid(), $1, 'nav_config', $2, 'JSON', now())
		ON CONFLICT (organization_id, key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = now()
	`, orgID, string(navJSON))
	if err != nil {
		writeJSONErr(w, "failed to update nav", http.StatusInternalServerError)
		return
	}

	h.cache.Invalidate()
	cfg, err := h.loadCached(r.Context())
	if err != nil {
		writeJSONErr(w, "failed to reload config", http.StatusInternalServerError)
		return
	}
	writeJSON(w, cfg.Nav)
}

func (h *Handler) PatchFeatures(w http.ResponseWriter, r *http.Request) {
	var features FeatureFlags
	if err := json.NewDecoder(r.Body).Decode(&features); err != nil {
		writeJSONErr(w, "invalid request body", http.StatusBadRequest)
		return
	}

	orgID := app.OrgID
	flagMap := map[string]bool{
		"feat_green_noting": features.GreenNotingEnabled,
		"feat_smtp":         features.ExternalSmtpEnabled,
		"feat_2fa_admin":    features.Require2faForAdmin,
	}
	for k, v := range flagMap {
		val := "false"
		if v {
			val = "true"
		}
		_, err := h.db.ExecContext(r.Context(), `
			INSERT INTO system_settings (id, organization_id, key, value, value_type, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, 'BOOLEAN', now())
			ON CONFLICT (organization_id, key) DO UPDATE
			SET value = EXCLUDED.value, updated_at = now()
		`, orgID, k, val)
		if err != nil {
			writeJSONErr(w, "failed to update features", http.StatusInternalServerError)
			return
		}
	}

	h.cache.Invalidate()
	cfg, err := h.loadCached(r.Context())
	if err != nil {
		writeJSONErr(w, "failed to reload config", http.StatusInternalServerError)
		return
	}
	writeJSON(w, cfg.Features)
}

func (h *Handler) PatchOrg(w http.ResponseWriter, r *http.Request) {
	var input OrgProfile
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONErr(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// organizations table only has name (see migration 02 — no timezone/locale/contact_email columns).
	_, err := h.db.ExecContext(r.Context(), `
		UPDATE organizations SET
			name       = COALESCE(NULLIF($1, ''), name),
			updated_at = now()
	`, input.Name)
	if err != nil {
		writeJSONErr(w, "failed to update org", http.StatusInternalServerError)
		return
	}

	h.cache.Invalidate()
	cfg, err := h.loadCached(r.Context())
	if err != nil {
		writeJSONErr(w, "failed to reload config", http.StatusInternalServerError)
		return
	}
	writeJSON(w, cfg.Org)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *Handler) loadCached(ctx context.Context) (*OrgConfig, error) {
	if cfg, ok := h.cache.Get(); ok {
		return cfg, nil
	}
	cfg, err := LoadOrgConfig(ctx, h.db)
	if err != nil {
		return nil, err
	}
	h.cache.Set(cfg)
	return cfg, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONErr(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// nullIfEmpty returns the string or empty string (used for COALESCE/NULLIF patterns).
func nullIfEmpty(s string) string {
	return s
}

// orgIDFromCtx is kept for compatibility with any future callers; unused by this handler.
func orgIDFromCtx(r *http.Request) (uuid.UUID, bool) {
	v := r.Context().Value(OrgIDContextKey)
	if v == nil {
		return uuid.UUID{}, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}
