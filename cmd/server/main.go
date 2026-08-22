// Comando server: la API HTTP en vivo. Nunca ejecuta chromedp (ver
// internal/preview/screenshot.go y cmd/shooter) — por diseño, para poder
// desplegarse en un free tier serverless (Cloud Run) sin empaquetar un
// navegador.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/config"
	"github.com/RenatoMart/repo-preview-service/internal/ghclient"
	"github.com/RenatoMart/repo-preview-service/internal/httpapi"
	"github.com/RenatoMart/repo-preview-service/internal/meta"
	"github.com/RenatoMart/repo-preview-service/internal/preview"
	"github.com/RenatoMart/repo-preview-service/internal/refresh"
	"github.com/RenatoMart/repo-preview-service/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cat, err := catalog.Load(cfg.CatalogPath)
	if err != nil {
		return err
	}

	st := store.New(cfg.DataDir)
	if err := st.Load(); err != nil {
		return err
	}

	gh := ghclient.New(cfg.GitHubToken)
	metaSvc := meta.NewService(gh)

	// Orden de la cascada: la captura ya commiteada por cmd/shooter
	// primero (si el proyecto tiene web desplegada), luego la social
	// preview propia del repo, luego la mejor imagen del README, y por
	// último la tarjeta generada, que nunca falla.
	resolver := preview.NewResolver(
		preview.NewCommittedScreenshotSource(cfg.ScreenshotsRepo, cfg.ScreenshotsBranch),
		preview.NewSocialPreviewSource(gh),
		preview.NewReadmeImageSource(),
		preview.NewCardSource(),
	)

	ref := refresh.New(cat, metaSvc, resolver, st, cfg.RefreshInterval)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	ref.Start(ctx)

	router := httpapi.NewRouter(cfg, cat, st, ref)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("servidor escuchando", "port", cfg.Port, "projects", cat.Len())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("apagando servidor")
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
