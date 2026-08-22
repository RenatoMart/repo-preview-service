// Package ghclient envuelve go-github con un transport que cachea
// respuestas por ETag (ver etag.go), para que verificar si un repo cambió
// no consuma la cuota de peticiones de la API de GitHub.
package ghclient

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v75/github"
)

// Client es un wrapper de alto nivel sobre go-github con los pocos
// métodos que este servicio necesita.
type Client struct {
	gh *github.Client
}

// New crea un Client. Si token está vacío, las peticiones se hacen sin
// autenticar (límite de 60 peticiones/hora en vez de 5000).
func New(token string) *Client {
	httpClient := &http.Client{
		Transport: newETagTransport(http.DefaultTransport),
		Timeout:   15 * time.Second,
	}

	gh := github.NewClient(httpClient)
	if token != "" {
		gh = gh.WithAuthToken(token)
	}

	return &Client{gh: gh}
}

// RepoInfo son los campos del repositorio que le importan al servicio.
type RepoInfo struct {
	DefaultBranch string
	PushedAt      time.Time
	Description   string
	HTMLURL       string
}

// Repository trae la información básica del repo. Es la llamada barata
// que se usa para decidir si hace falta refrescar el resto (ver
// internal/refresh): con el ETag transport, si nada cambió esta llamada
// no consume cuota.
func (c *Client) Repository(ctx context.Context, owner, name string) (RepoInfo, error) {
	repo, _, err := c.gh.Repositories.Get(ctx, owner, name)
	if err != nil {
		return RepoInfo{}, fmt.Errorf("ghclient: GET repo %s/%s: %w", owner, name, err)
	}

	info := RepoInfo{
		DefaultBranch: repo.GetDefaultBranch(),
		Description:   repo.GetDescription(),
		HTMLURL:       repo.GetHTMLURL(),
	}
	if repo.PushedAt != nil {
		info.PushedAt = repo.PushedAt.Time
	}
	if info.DefaultBranch == "" {
		info.DefaultBranch = "main"
	}
	return info, nil
}

// Readme trae el contenido decodificado del README del repo.
func (c *Client) Readme(ctx context.Context, owner, name string) (string, error) {
	content, _, err := c.gh.Repositories.GetReadme(ctx, owner, name, nil)
	if err != nil {
		return "", fmt.Errorf("ghclient: GET readme %s/%s: %w", owner, name, err)
	}
	raw, err := content.GetContent()
	if err != nil {
		return "", fmt.Errorf("ghclient: decodificando readme %s/%s: %w", owner, name, err)
	}
	return raw, nil
}

// Languages trae los lenguajes del repo como porcentajes (0-100), sobre
// el total de bytes reportado por GitHub.
func (c *Client) Languages(ctx context.Context, owner, name string) (map[string]float64, error) {
	langs, _, err := c.gh.Repositories.ListLanguages(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("ghclient: GET languages %s/%s: %w", owner, name, err)
	}

	var total int
	for _, bytes := range langs {
		total += bytes
	}
	if total == 0 {
		return map[string]float64{}, nil
	}

	pct := make(map[string]float64, len(langs))
	for lang, bytes := range langs {
		pct[lang] = float64(bytes) / float64(total) * 100
	}
	return pct, nil
}

// OGImage intenta leer el meta tag og:image de la página del repo en
// github.com, para detectar si tiene una "social preview" personalizada
// subida a mano. Ver internal/preview/social.go, que interpreta el
// resultado.
func (c *Client) OGImage(ctx context.Context, owner, name string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/%s", owner, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.gh.Client().Do(req)
	if err != nil {
		return "", fmt.Errorf("ghclient: GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ghclient: GET %s: status %d", url, resp.StatusCode)
	}

	return extractOGImage(resp.Body)
}
