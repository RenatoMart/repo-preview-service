// Comando shooter: toma capturas de pantalla reales con chromedp para
// los proyectos con web desplegada, y las deja en data/screenshots/.
//
// Deliberadamente NO es parte del servidor en vivo (cmd/server): corre
// como un paso de un workflow de GitHub Actions (que trae Chrome
// preinstalado y es gratis para repos públicos), disparado por push o
// por un cron de respaldo. El PNG resultante se commitea al repo y
// jsDelivr lo sirve de ahí en adelante como CDN gratis — el servidor en
// vivo nunca necesita abrir un navegador (ver
// internal/preview/committed.go).
//
// Reutiliza la misma idea de pushed_at que internal/refresh: guarda un
// sidecar JSON junto a cada PNG con la fecha del último push capturado,
// y solo vuelve a abrir Chrome si esa fecha cambió (o si se pasa
// -force).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/ghclient"
	"github.com/RenatoMart/repo-preview-service/internal/meta"
	"github.com/RenatoMart/repo-preview-service/internal/preview"
)

func main() {
	catalogPath := flag.String("catalog", "./configs/projects.yaml", "ruta al catálogo de proyectos")
	dataDir := flag.String("data", "./data", "directorio de salida (se escribe en <data>/screenshots/)")
	onlySlug := flag.String("slug", "", "si se da, captura solo este proyecto")
	chromePath := flag.String("chrome-path", os.Getenv("CHROME_PATH"), "ruta al binario de Chrome/Chromium (vacío = autodetectar)")
	force := flag.Bool("force", false, "capturar aunque pushed_at no haya cambiado")
	flag.Parse()

	if err := run(*catalogPath, *dataDir, *onlySlug, *chromePath, *force); err != nil {
		slog.Error("cmd/shooter: fallo", "err", err)
		os.Exit(1)
	}
}

func run(catalogPath, dataDir, onlySlug, chromePath string, force bool) error {
	cat, err := catalog.Load(catalogPath)
	if err != nil {
		return err
	}

	targets := cat.All()
	if onlySlug != "" {
		p, ok := cat.BySlug(onlySlug)
		if !ok {
			return fmt.Errorf("cmd/shooter: slug %q no está en el catálogo", onlySlug)
		}
		targets = []catalog.Project{p}
	}

	screenshotsDir := filepath.Join(dataDir, "screenshots")
	if err := os.MkdirAll(screenshotsDir, 0o755); err != nil {
		return err
	}

	gh := ghclient.New(os.Getenv("GITHUB_TOKEN"))
	metaSvc := meta.NewService(gh)
	shot := preview.NewScreenshotSource(chromePath)

	ctx := context.Background()
	var updated int
	for _, p := range targets {
		if !p.HasLive() {
			continue
		}

		didUpdate, err := shootOne(ctx, metaSvc, shot, screenshotsDir, p, force)
		if err != nil {
			// Un proyecto que falla no debe abortar el resto: se
			// registra y se sigue con los demás.
			slog.Error("cmd/shooter: capturando proyecto", "slug", p.Slug, "err", err)
			continue
		}
		if didUpdate {
			updated++
		}
	}

	slog.Info("cmd/shooter: listo", "actualizados", updated, "revisados", len(targets))
	return nil
}

// sidecar guarda, junto al PNG, la fecha de push que se capturó, para
// decidir en la próxima corrida si hace falta abrir Chrome de nuevo.
type sidecar struct {
	PushedAt time.Time `json:"pushedAt"`
}

func sidecarPath(dir, slug string) string {
	return filepath.Join(dir, slug+".json")
}

func shootOne(ctx context.Context, metaSvc *meta.Service, shot *preview.ScreenshotSource, dir string, p catalog.Project, force bool) (bool, error) {
	info, err := metaSvc.RepoInfo(ctx, p)
	if err != nil {
		return false, fmt.Errorf("repo info: %w", err)
	}

	if !force {
		if prev, ok := readSidecar(dir, p.Slug); ok && prev.PushedAt.Equal(info.PushedAt) {
			slog.Info("cmd/shooter: sin cambios, se omite", "slug", p.Slug)
			return false, nil
		}
	}

	slog.Info("cmd/shooter: capturando", "slug", p.Slug, "url", p.Live)
	img, err := shot.Render(ctx, p, meta.Metadata{})
	if err != nil {
		return false, fmt.Errorf("captura: %w", err)
	}

	pngPath := filepath.Join(dir, p.Slug+".png")
	if err := os.WriteFile(pngPath, img.Bytes, 0o644); err != nil {
		return false, fmt.Errorf("guardando png: %w", err)
	}
	if err := writeSidecar(dir, p.Slug, sidecar{PushedAt: info.PushedAt}); err != nil {
		return false, fmt.Errorf("guardando sidecar: %w", err)
	}

	slog.Info("cmd/shooter: capturado", "slug", p.Slug, "bytes", len(img.Bytes))
	return true, nil
}

func readSidecar(dir, slug string) (sidecar, bool) {
	raw, err := os.ReadFile(sidecarPath(dir, slug))
	if err != nil {
		return sidecar{}, false
	}
	var s sidecar
	if err := json.Unmarshal(raw, &s); err != nil {
		return sidecar{}, false
	}
	return s, true
}

func writeSidecar(dir, slug string, s sidecar) error {
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sidecarPath(dir, slug), raw, 0o644)
}
