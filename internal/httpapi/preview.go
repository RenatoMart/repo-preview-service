package httpapi

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/meta"
	"github.com/RenatoMart/repo-preview-service/internal/preview"
	"github.com/RenatoMart/repo-preview-service/internal/store"
)

// PreviewHandler sirve la imagen de previsualización de un proyecto.
type PreviewHandler struct {
	catalog *catalog.Catalog
	store   *store.Store
	card    *preview.CardSource
}

// NewPreviewHandler crea el handler.
func NewPreviewHandler(cat *catalog.Catalog, st *store.Store) *PreviewHandler {
	return &PreviewHandler{catalog: cat, store: st, card: preview.NewCardSource()}
}

// Get responde GET /api/v1/projects/{slug}/preview.
//
// Si internal/refresh todavía no terminó su primer ciclo para este slug
// (arranque reciente, o el proyecto es nuevo), no se devuelve un error:
// se genera la tarjeta al vuelo (es local, sin red, así que es barato) en
// vez de dejar al frontend con una imagen rota.
func (h *PreviewHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	p, ok := h.catalog.BySlug(slug)
	if !ok {
		writeError(w, http.StatusNotFound, "proyecto no encontrado")
		return
	}

	e, hasEntry := h.store.Get(slug)
	img := e.Preview
	pushedAt := e.Meta.PushedAt

	if !hasEntry || len(img.Bytes) == 0 {
		fallback, err := h.card.Render(r.Context(), p, meta.Metadata{})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "no se pudo generar la previsualización")
			return
		}
		img = fallback
	}

	etag := fmt.Sprintf(`"%s-%d-%s"`, slug, pushedAt.Unix(), img.Source)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=86400")

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", img.ContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(img.Bytes)
}
