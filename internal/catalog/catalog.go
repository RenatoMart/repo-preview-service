// Package catalog es la fuente de verdad de qué proyectos expone el
// servicio. Cargar la lista desde YAML (en vez de aceptar repos/URLs
// arbitrarios en las peticiones) es lo que evita que el endpoint de
// captura pueda usarse para SSRF: solo se puede pedir preview de un slug
// que exista aquí, y la URL a capturar la decide este archivo, no el
// llamante.
package catalog

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Project es una entrada estática del catálogo. Los datos dinámicos
// (README, lenguajes, imagen) se resuelven aparte, en internal/meta y
// internal/preview, y se combinan con esto en internal/httpapi.
type Project struct {
	Slug        string   `yaml:"slug"`
	Repo        string   `yaml:"repo"` // "owner/name"
	Title       string   `yaml:"title"`
	Category    string   `yaml:"category"`
	Live        string   `yaml:"live"`
	Accent      string   `yaml:"accent"`
	Tags        []string `yaml:"tags"`
	Description string   `yaml:"description"`
}

// Owner devuelve la parte "owner" de Repo.
func (p Project) Owner() string {
	owner, _, _ := strings.Cut(p.Repo, "/")
	return owner
}

// Name devuelve la parte "name" de Repo.
func (p Project) Name() string {
	_, name, _ := strings.Cut(p.Repo, "/")
	return name
}

// HasLive indica si el proyecto tiene una URL desplegada capturable.
func (p Project) HasLive() bool {
	return p.Live != ""
}

// Catalog es la lista de proyectos ya validada, indexada para lookups
// rápidos por slug y por repo.
type Catalog struct {
	projects []Project
	bySlug   map[string]Project
	byRepo   map[string]Project // repo en minúsculas -> Project
}

var repoPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*/[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Load lee y valida el catálogo desde un archivo YAML. Un catálogo mal
// formado (slugs duplicados, repo con formato inválido, live que no es
// https) hace fallar el arranque en vez de fallar más tarde en runtime.
func Load(path string) (*Catalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("catalog: leyendo %s: %w", path, err)
	}

	var projects []Project
	if err := yaml.Unmarshal(raw, &projects); err != nil {
		return nil, fmt.Errorf("catalog: parseando %s: %w", path, err)
	}
	if len(projects) == 0 {
		return nil, fmt.Errorf("catalog: %s no define ningún proyecto", path)
	}

	c := &Catalog{
		projects: projects,
		bySlug:   make(map[string]Project, len(projects)),
		byRepo:   make(map[string]Project, len(projects)),
	}

	for i, p := range projects {
		if p.Slug == "" {
			return nil, fmt.Errorf("catalog: entrada #%d sin slug", i)
		}
		if _, dup := c.bySlug[p.Slug]; dup {
			return nil, fmt.Errorf("catalog: slug %q duplicado", p.Slug)
		}
		if !repoPattern.MatchString(p.Repo) {
			return nil, fmt.Errorf("catalog: %s: repo %q no tiene formato owner/name", p.Slug, p.Repo)
		}
		if p.Live != "" && !strings.HasPrefix(p.Live, "https://") {
			return nil, fmt.Errorf("catalog: %s: live %q debe ser una URL https", p.Slug, p.Live)
		}
		if p.Title == "" {
			return nil, fmt.Errorf("catalog: %s: title vacío", p.Slug)
		}

		c.bySlug[p.Slug] = p
		c.byRepo[strings.ToLower(p.Repo)] = p
	}

	return c, nil
}

// All devuelve todos los proyectos, en el orden del YAML.
func (c *Catalog) All() []Project {
	out := make([]Project, len(c.projects))
	copy(out, c.projects)
	return out
}

// BySlug busca un proyecto por su slug.
func (c *Catalog) BySlug(slug string) (Project, bool) {
	p, ok := c.bySlug[slug]
	return p, ok
}

// ByRepo busca un proyecto por "owner/name" (case-insensitive, como
// llegan los nombres en los payloads de webhook de GitHub).
func (c *Catalog) ByRepo(repo string) (Project, bool) {
	p, ok := c.byRepo[strings.ToLower(repo)]
	return p, ok
}

// Len devuelve el número de proyectos en el catálogo.
func (c *Catalog) Len() int {
	return len(c.projects)
}
