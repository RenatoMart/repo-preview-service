package preview

import (
	"context"
	"net/http"
	"time"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/meta"
)

// ReadmeImageSource sirve la mejor imagen encontrada dentro del README
// del proyecto (ver internal/meta/readme.go para la extracción, el
// filtrado de badges, y la elección de la mejor candidata).
type ReadmeImageSource struct {
	client *http.Client
}

// NewReadmeImageSource crea la fuente.
func NewReadmeImageSource() *ReadmeImageSource {
	return &ReadmeImageSource{client: &http.Client{Timeout: 8 * time.Second}}
}

func (s *ReadmeImageSource) Name() string { return "readme" }

func (s *ReadmeImageSource) Render(ctx context.Context, _ catalog.Project, m meta.Metadata) (Image, error) {
	best, ok := meta.PickBestImage(m.Images)
	if !ok {
		return Image{}, ErrNotApplicable
	}
	return fetchImage(ctx, s.client, best.URL)
}
