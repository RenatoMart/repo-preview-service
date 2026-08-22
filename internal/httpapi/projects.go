package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/store"
)

// projectDTO es la forma pública de un proyecto, combinando el catálogo
// estático (configs/projects.yaml) con lo que internal/refresh ya haya
// resuelto (README, lenguajes, imagen). Antes del primer refresco los
// campos dinámicos simplemente van vacíos.
type projectDTO struct {
	Slug          string             `json:"slug"`
	Title         string             `json:"title"`
	Category      string             `json:"category"`
	Description   string             `json:"description"`
	Tags          []string           `json:"tags"`
	Accent        string             `json:"accent"`
	RepoURL       string             `json:"repoUrl"`
	LiveURL       *string            `json:"liveUrl"`
	ReadmeHTML    string             `json:"readmeHtml,omitempty"`
	Languages     map[string]float64 `json:"languages,omitempty"`
	PushedAt      *time.Time         `json:"pushedAt,omitempty"`
	PreviewURL    string             `json:"previewUrl"`
	PreviewSource string             `json:"previewSource,omitempty"`
}

func toDTO(p catalog.Project, e store.Entry, hasEntry bool) projectDTO {
	dto := projectDTO{
		Slug:        p.Slug,
		Title:       p.Title,
		Category:    p.Category,
		Description: p.Description,
		Tags:        p.Tags,
		Accent:      p.Accent,
		RepoURL:     "https://github.com/" + p.Repo,
		PreviewURL:  "/api/v1/projects/" + p.Slug + "/preview",
	}
	if p.HasLive() {
		live := p.Live
		dto.LiveURL = &live
	}
	if hasEntry {
		dto.ReadmeHTML = e.Meta.ReadmeHTML
		dto.Languages = e.Meta.Languages
		if !e.Meta.PushedAt.IsZero() {
			t := e.Meta.PushedAt
			dto.PushedAt = &t
		}
		dto.PreviewSource = e.Preview.Source
	}
	return dto
}

// ProjectsHandler sirve la lista y el detalle de proyectos.
type ProjectsHandler struct {
	catalog *catalog.Catalog
	store   *store.Store
}

// NewProjectsHandler crea el handler.
func NewProjectsHandler(cat *catalog.Catalog, st *store.Store) *ProjectsHandler {
	return &ProjectsHandler{catalog: cat, store: st}
}

// List responde GET /api/v1/projects.
func (h *ProjectsHandler) List(w http.ResponseWriter, _ *http.Request) {
	all := h.catalog.All()
	out := make([]projectDTO, 0, len(all))
	for _, p := range all {
		e, ok := h.store.Get(p.Slug)
		out = append(out, toDTO(p, e, ok))
	}
	writeJSON(w, http.StatusOK, out)
}

// Detail responde GET /api/v1/projects/{slug}.
func (h *ProjectsHandler) Detail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	p, ok := h.catalog.BySlug(slug)
	if !ok {
		writeError(w, http.StatusNotFound, "proyecto no encontrado")
		return
	}
	e, hasEntry := h.store.Get(slug)
	writeJSON(w, http.StatusOK, toDTO(p, e, hasEntry))
}
