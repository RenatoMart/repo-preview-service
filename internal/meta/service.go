// Package meta trae y procesa la información dinámica de un repo: README
// (renderizado a HTML + imágenes extraídas), lenguajes y la fecha de
// último push que usa internal/refresh para decidir si hace falta
// refrescar algo.
package meta

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/go-github/v75/github"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/ghclient"
)

// Metadata es toda la información dinámica de un proyecto, ya procesada
// y lista para combinarse con catalog.Project en la respuesta de la API.
type Metadata struct {
	ReadmeHTML    string
	Images        []ImageCandidate // imágenes del README, filtradas de badges
	Languages     map[string]float64
	PushedAt      time.Time
	DefaultBranch string
	Description   string
}

// Service orquesta las llamadas a GitHub necesarias para construir un Metadata.
type Service struct {
	gh *ghclient.Client
}

// NewService crea un Service sobre el cliente de GitHub dado.
func NewService(gh *ghclient.Client) *Service {
	return &Service{gh: gh}
}

// RepoInfo trae solo los datos básicos del repo (pushed_at, default
// branch). Es intencionalmente la única llamada que internal/refresh hace
// en cada ciclo para decidir si el resto del refresco es necesario: con
// el ETag transport, si nada cambió no consume cuota de la API.
func (s *Service) RepoInfo(ctx context.Context, p catalog.Project) (ghclient.RepoInfo, error) {
	return s.gh.Repository(ctx, p.Owner(), p.Name())
}

// Fetch trae README y lenguajes, y arma el Metadata completo. Es la parte
// cara de la actualización: solo debe llamarse cuando RepoInfo indica que
// pushed_at cambió respecto a lo que ya se tenía guardado.
func (s *Service) Fetch(ctx context.Context, p catalog.Project, info ghclient.RepoInfo) (Metadata, error) {
	raw, err := s.gh.Readme(ctx, p.Owner(), p.Name())
	if err != nil && !isNotFound(err) {
		return Metadata{}, fmt.Errorf("meta: readme de %s: %w", p.Repo, err)
	}

	html, renderErr := RenderHTML(raw)
	if renderErr != nil {
		return Metadata{}, renderErr
	}
	images := ExtractImages(raw, p.Owner(), p.Name(), info.DefaultBranch)

	langs, err := s.gh.Languages(ctx, p.Owner(), p.Name())
	if err != nil {
		return Metadata{}, fmt.Errorf("meta: languages de %s: %w", p.Repo, err)
	}

	return Metadata{
		ReadmeHTML:    html,
		Images:        images,
		Languages:     langs,
		PushedAt:      info.PushedAt,
		DefaultBranch: info.DefaultBranch,
		Description:   info.Description,
	}, nil
}

// isNotFound reporta si err es un 404 de la API de GitHub. Un repo sin
// README no debe hacer fallar todo el refresco, solo dejar ReadmeHTML
// vacío.
func isNotFound(err error) bool {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		return ghErr.Response != nil && ghErr.Response.StatusCode == 404
	}
	return false
}
