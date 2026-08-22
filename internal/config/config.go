// Package config carga la configuración del servicio desde variables de
// entorno, con valores por defecto razonables para desarrollo local.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config agrupa toda la configuración del servidor HTTP en vivo.
//
// Nota: este servidor nunca ejecuta chromedp. Las capturas de pantalla se
// generan aparte, en GitHub Actions, con la herramienta cmd/shooter (ver
// ese paquete), y se sirven desde el CDN de jsDelivr sobre este mismo
// repositorio. Eso es lo que permite desplegar el servidor en un free
// tier serverless (Cloud Run) sin empaquetar un navegador.
type Config struct {
	Port    string
	DataDir string

	CatalogPath string

	GitHubToken string

	GitHubWebhookSecret string
	AdminToken          string

	CORSAllowedOrigins []string

	RefreshInterval time.Duration

	// ScreenshotsRepo y ScreenshotsBranch apuntan al repo donde
	// cmd/shooter (vía GitHub Actions) commitea data/screenshots/{slug}.png.
	// Con esto se arma la URL del CDN de jsDelivr que sirve la fuente
	// "committed" de la cascada (ver internal/preview/committed.go).
	ScreenshotsRepo   string
	ScreenshotsBranch string
}

// Load lee la configuración desde el entorno. No falla si faltan valores
// opcionales (token de GitHub, secretos); esos casos se degradan en
// runtime (menor límite de peticiones, webhook deshabilitado, etc).
func Load() (Config, error) {
	cfg := Config{
		Port:                getEnv("PORT", "8080"),
		DataDir:             getEnv("DATA_DIR", "./data"),
		CatalogPath:         getEnv("CATALOG_PATH", "./configs/projects.yaml"),
		GitHubToken:         os.Getenv("GITHUB_TOKEN"),
		GitHubWebhookSecret: os.Getenv("GITHUB_WEBHOOK_SECRET"),
		AdminToken:          os.Getenv("ADMIN_TOKEN"),
		CORSAllowedOrigins:  splitAndTrim(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		ScreenshotsRepo:     getEnv("SCREENSHOTS_REPO", "RenatoMart/repo-preview-service"),
		ScreenshotsBranch:   getEnv("SCREENSHOTS_BRANCH", "main"),
	}

	var err error
	if cfg.RefreshInterval, err = getDuration("REFRESH_INTERVAL", 30*time.Minute); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s inválido (%q): %w", key, v, err)
	}
	return d, nil
}

func splitAndTrim(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
