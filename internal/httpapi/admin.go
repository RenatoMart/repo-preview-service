package httpapi

import (
	"crypto/subtle"
	"net/http"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/refresh"
)

// AdminHandler permite forzar un refresco manual, protegido por un
// token compartido (no por sesión de usuario: este servicio no tiene
// usuarios, solo lo llama su dueño desde la terminal).
type AdminHandler struct {
	token     string
	catalog   *catalog.Catalog
	refresher *refresh.Refresher
}

// NewAdminHandler crea el handler. Si token está vacío, el endpoint
// rechaza todas las peticiones.
func NewAdminHandler(token string, cat *catalog.Catalog, r *refresh.Refresher) *AdminHandler {
	return &AdminHandler{token: token, catalog: cat, refresher: r}
}

func (h *AdminHandler) authorized(r *http.Request) bool {
	if h.token == "" {
		return false
	}
	got := r.Header.Get("X-Admin-Token")
	return subtle.ConstantTimeCompare([]byte(got), []byte(h.token)) == 1
}

// Refresh responde POST /api/v1/admin/refresh[?slug=...]. Sin slug,
// encola un refresco de todo el catálogo.
func (h *AdminHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r) {
		writeError(w, http.StatusUnauthorized, "no autorizado")
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		for _, p := range h.catalog.All() {
			h.refresher.Enqueue(p.Slug)
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "refresco encolado para todos los proyectos"})
		return
	}

	if _, ok := h.catalog.BySlug(slug); !ok {
		writeError(w, http.StatusNotFound, "proyecto no encontrado")
		return
	}
	h.refresher.Enqueue(slug)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "refresco encolado", "slug": slug})
}
