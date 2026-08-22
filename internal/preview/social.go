package preview

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/ghclient"
	"github.com/RenatoMart/repo-preview-service/internal/meta"
)

// SocialPreviewSource detecta si el repo tiene una "social preview" subida
// a mano (GitHub → Settings → Social preview) y, si es así, la usa.
//
// La API REST no expone si un repo tiene una imagen propia; se detecta
// leyendo el meta tag og:image de la página del repo en github.com: si
// apunta a repository-images.githubusercontent.com es una imagen que tú
// subiste, si apunta a opengraph.githubassets.com es la tarjeta genérica
// autogenerada y se descarta para que la cascada siga con la siguiente
// fuente.
type SocialPreviewSource struct {
	gh     *ghclient.Client
	client *http.Client
}

// NewSocialPreviewSource crea la fuente.
func NewSocialPreviewSource(gh *ghclient.Client) *SocialPreviewSource {
	return &SocialPreviewSource{gh: gh, client: &http.Client{Timeout: 8 * time.Second}}
}

func (s *SocialPreviewSource) Name() string { return "social" }

func (s *SocialPreviewSource) Render(ctx context.Context, p catalog.Project, _ meta.Metadata) (Image, error) {
	ogImage, err := s.gh.OGImage(ctx, p.Owner(), p.Name())
	if err != nil {
		if errors.Is(err, ghclient.ErrOGImageNotFound) {
			return Image{}, ErrNotApplicable
		}
		return Image{}, err
	}
	if !strings.Contains(ogImage, "repository-images.githubusercontent.com") {
		return Image{}, ErrNotApplicable // tarjeta genérica, no una imagen propia
	}

	return fetchImage(ctx, s.client, ogImage)
}
