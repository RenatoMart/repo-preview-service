// Package preview implementa la cascada de previsualización: varias
// fuentes de imagen, probadas en orden, de forma que cada proyecto recibe
// automáticamente la mejor disponible y mejora sola según se sube más
// material a GitHub (ver internal/preview/source.go y el plan del
// servicio para el detalle de las cuatro fuentes).
package preview

import (
	"context"
	"errors"
	"log/slog"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/meta"
)

// ErrNotApplicable indica que una fuente no tiene nada que ofrecer para
// este proyecto concreto (no tiene web desplegada, el README no tiene
// imágenes, etc.), y el resolver debe seguir con la siguiente de la
// cascada. Cualquier otro error se trata como un fallo real de la fuente
// (red, timeout) y también se pasa a la siguiente, pero se registra.
var ErrNotApplicable = errors.New("preview: fuente no aplicable")

// Image es el resultado de resolver la cascada para un proyecto.
type Image struct {
	Bytes       []byte
	ContentType string
	Source      string // nombre de la fuente que la produjo (screenshot/social/readme/card)
}

// Source es una fuente de imagen de previsualización.
type Source interface {
	Name() string
	Render(ctx context.Context, p catalog.Project, m meta.Metadata) (Image, error)
}

// Resolver prueba las fuentes en el orden dado y devuelve la primera que
// produzca una imagen. La última fuente de la lista debe ser una que
// nunca falle (en este servicio, CardSource), para que Resolve siempre
// tenga algo que devolver.
type Resolver struct {
	sources []Source
}

// NewResolver arma un resolver con las fuentes en orden de prioridad.
func NewResolver(sources ...Source) *Resolver {
	return &Resolver{sources: sources}
}

// Resolve recorre las fuentes en orden y devuelve la primera imagen
// disponible.
func (r *Resolver) Resolve(ctx context.Context, p catalog.Project, m meta.Metadata) (Image, error) {
	for _, s := range r.sources {
		img, err := s.Render(ctx, p, m)
		if err == nil {
			img.Source = s.Name()
			return img, nil
		}
		if !errors.Is(err, ErrNotApplicable) {
			slog.WarnContext(ctx, "preview: fuente falló, se prueba la siguiente",
				"slug", p.Slug, "source", s.Name(), "err", err)
		}
	}
	return Image{}, errors.New("preview: ninguna fuente disponible (falta registrar un fallback que nunca falle)")
}
