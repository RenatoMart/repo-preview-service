package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/config"
	"github.com/RenatoMart/repo-preview-service/internal/refresh"
	"github.com/RenatoMart/repo-preview-service/internal/store"
)

// NewRouter arma el router HTTP del servicio.
func NewRouter(cfg config.Config, cat *catalog.Catalog, st *store.Store, ref *refresh.Refresher) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))
	r.Use(requestLogger)

	// Orígenes explícitos, sin "*": el endpoint de admin no debe ser
	// invocable desde cualquier página que el navegador de alguien tenga
	// abierta.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders: []string{"Content-Type", "X-Admin-Token", "X-Hub-Signature-256", "X-GitHub-Event", "If-None-Match"},
		MaxAge:         300,
	}))

	// UptimeRobot y monitores similares chequean con HEAD, no GET, para
	// ahorrar ancho de banda — sin este registro, chi responde 405 (solo
	// enruta el método exacto que se registró).
	r.Get("/healthz", healthHandler(cat))
	r.Head("/healthz", healthHandler(cat))

	projects := NewProjectsHandler(cat, st)
	previewH := NewPreviewHandler(cat, st)
	webhook := NewWebhookHandler(cfg.GitHubWebhookSecret, cat, ref)
	admin := NewAdminHandler(cfg.AdminToken, cat, ref)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/projects", projects.List)
		r.Get("/projects/{slug}", projects.Detail)
		r.Get("/projects/{slug}/preview", previewH.Get)
		r.Post("/webhooks/github", webhook.Handle)
		r.Post("/admin/refresh", admin.Refresh)
	})

	return r
}

func healthHandler(cat *catalog.Catalog) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "projects": cat.Len()})
	}
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
