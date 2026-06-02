package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// OrgConfig mirrors the AppRuntimeConfig shape expected by the Nuxt frontend.
type OrgConfig struct {
	Branding BrandingConfig `json:"branding"`
	Nav      []NavGroup     `json:"nav"`
	Features FeatureFlags   `json:"features"`
	Org      OrgProfile     `json:"org"`
}

type BrandingConfig struct {
	PrimaryColor   string `json:"primaryColor"`
	NeutralPalette string `json:"neutralPalette"`
	FontFamily     string `json:"fontFamily"`
	Wordmark       string `json:"wordmark"`
	LogoPath       string `json:"logoPath"`
}

type NavGroup struct {
	Label string    `json:"label"`
	Roles []string  `json:"roles"`
	Items []NavItem `json:"items"`
}

type NavItem struct {
	Label string `json:"label"`
	Icon  string `json:"icon"`
	To    string `json:"to"`
	Badge *int   `json:"badge"`
}

type FeatureFlags struct {
	GreenNotingEnabled  bool `json:"greenNotingEnabled"`
	ExternalSmtpEnabled bool `json:"externalSmtpEnabled"`
	Require2faForAdmin  bool `json:"require2faForAdmin"`
}

type OrgProfile struct {
	Name         string `json:"name"`
	Timezone     string `json:"timezone"`
	Locale       string `json:"locale"`
	ContactEmail string `json:"contactEmail"`
}

// LoadOrgConfig reads the current installation config from the database.
// Single-tenant: no org filter — there is exactly one org per installation.
// Returns safe defaults for any missing rows.
func LoadOrgConfig(ctx context.Context, db *sql.DB) (*OrgConfig, error) {
	cfg := &OrgConfig{
		Branding: BrandingConfig{
			PrimaryColor:   "#4f46e5",
			NeutralPalette: "slate",
			FontFamily:     "Inter",
			Wordmark:       "Aethel Workspace",
		},
		Features: FeatureFlags{},
		Org: OrgProfile{
			Timezone: "UTC",
			Locale:   "en-US",
		},
	}

	// Load branding — branding_configs has one row per installation.
	row := db.QueryRowContext(ctx, `
		SELECT
			COALESCE(primary_brand_color, '#4f46e5'),
			COALESCE(neutral_palette, 'slate'),
			COALESCE(font_family, 'Inter'),
			COALESCE(wordmark, 'Aethel Workspace'),
			COALESCE(logo_file_path, '')
		FROM branding_configs
		LIMIT 1
	`)
	var logoPath string
	err := row.Scan(
		&cfg.Branding.PrimaryColor,
		&cfg.Branding.NeutralPalette,
		&cfg.Branding.FontFamily,
		&cfg.Branding.Wordmark,
		&logoPath,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("load branding config: %w", err)
	}
	cfg.Branding.LogoPath = logoPath

	// Load nav config from system_settings (key/value store).
	var navJSON sql.NullString
	err = db.QueryRowContext(ctx, `
		SELECT value
		FROM system_settings
		WHERE key = 'nav_config'
		LIMIT 1
	`).Scan(&navJSON)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("load nav config: %w", err)
	}
	if navJSON.Valid && navJSON.String != "" {
		_ = json.Unmarshal([]byte(navJSON.String), &cfg.Nav)
	}

	// Load feature flags from system_settings.
	featureRows, err := db.QueryContext(ctx, `
		SELECT key, value
		FROM system_settings
		WHERE key IN ('feat_green_noting', 'feat_smtp', 'feat_2fa_admin')
	`)
	if err == nil {
		defer featureRows.Close()
		for featureRows.Next() {
			var k, v string
			if scanErr := featureRows.Scan(&k, &v); scanErr == nil {
				switch k {
				case "feat_green_noting":
					cfg.Features.GreenNotingEnabled = v == "true"
				case "feat_smtp":
					cfg.Features.ExternalSmtpEnabled = v == "true"
				case "feat_2fa_admin":
					cfg.Features.Require2faForAdmin = v == "true"
				}
			}
		}
	}

	// Load org name from organizations (single row per installation).
	err = db.QueryRowContext(ctx, `
		SELECT COALESCE(name, '')
		FROM organizations
		LIMIT 1
	`).Scan(&cfg.Org.Name)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("load org profile: %w", err)
	}

	return cfg, nil
}
