// Package docs serves the OpenAPI 3.1 specification and the Scalar interactive
// API explorer. It has no knowledge of domain types, services, or repositories.
// Its only dependency is the embedded spec and HTML files.
package docs

import (
	"context"
	_ "embed"
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
)

//go:embed openapi.yaml
var specBytes []byte

//go:embed scalar.html
var scalarHTML []byte

// ValidateSpec parses and validates the embedded OpenAPI spec at startup.
// Panics on any validation error — an invalid spec is a programmer error,
// not a recoverable runtime condition. Call this once from main.go before
// registering routes.
func ValidateSpec() {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(specBytes)
	if err != nil {
		panic("openapi: failed to parse spec: " + err.Error())
	}
	if err := doc.Validate(context.Background()); err != nil {
		panic("openapi: spec validation failed: " + err.Error())
	}
}

// Enabled reports whether the API docs route should be registered.
// Returns false when AETHEL_DISABLE_API_DOCS=true is set.
// Enabled by default; self-hosted operators control network-level access.
func Enabled() bool {
	return os.Getenv("AETHEL_DISABLE_API_DOCS") != "true"
}

// Handler returns an http.Handler that serves the Scalar UI and the raw
// OpenAPI YAML spec. Register this under /api/docs in the chi router.
//
// Routes served:
//
//	GET /api/docs              → Scalar UI (HTML)
//	GET /api/docs/openapi.yaml → Raw OpenAPI 3.1 spec (YAML)
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Match the full path to avoid prefix-stripping ambiguity with chi Mount.
		switch {
		case r.URL.Path == "/api/docs/openapi.yaml":
			w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=300")
			_, _ = w.Write(specBytes)
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			_, _ = w.Write(scalarHTML)
		}
	})
}
