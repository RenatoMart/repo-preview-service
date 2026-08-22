package preview

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/meta"
)

// CommittedScreenshotSource sirve capturas ya generadas fuera de este
// servidor: cmd/shooter, corriendo en GitHub Actions (que trae Chrome
// preinstalado y es gratis para repos públicos), las deja guardadas en
// data/screenshots/{slug}.png dentro de este mismo repo cuando detecta un
// push nuevo en un proyecto con web desplegada. jsDelivr sirve esos
// archivos como CDN gratuito directamente desde GitHub.
//
// El servidor en vivo nunca abre un navegador: esta fuente solo pregunta
// "¿existe esa URL?". Es lo que permite desplegarlo en un free tier
// serverless sin empaquetar Chrome.
type CommittedScreenshotSource struct {
	client *http.Client
	repo   string // "owner/name" de este mismo repo
	branch string
}

// NewCommittedScreenshotSource crea la fuente. repo y branch identifican
// dónde cmd/shooter commitea las capturas (ver internal/config).
func NewCommittedScreenshotSource(repo, branch string) *CommittedScreenshotSource {
	return &CommittedScreenshotSource{
		client: &http.Client{Timeout: 5 * time.Second},
		repo:   repo,
		branch: branch,
	}
}

func (s *CommittedScreenshotSource) Name() string { return "screenshot" }

func (s *CommittedScreenshotSource) Render(ctx context.Context, p catalog.Project, _ meta.Metadata) (Image, error) {
	if !p.HasLive() {
		return Image{}, ErrNotApplicable
	}

	url := fmt.Sprintf("https://cdn.jsdelivr.net/gh/%s@%s/data/screenshots/%s.png", s.repo, s.branch, p.Slug)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Image{}, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return Image{}, fmt.Errorf("preview: consultando captura de %s: %w", p.Slug, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return Image{}, ErrNotApplicable
	}
	if resp.StatusCode != http.StatusOK {
		return Image{}, fmt.Errorf("preview: captura de %s: status %d", p.Slug, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Image{}, err
	}
	return Image{Bytes: body, ContentType: "image/png"}, nil
}
