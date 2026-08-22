package preview

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/meta"
)

// ScreenshotSource toma una captura real de la web desplegada de un
// proyecto con chromedp (un Chrome real, sin cabeza).
//
// Deliberadamente NO se usa en el servidor HTTP en vivo (ver
// internal/httpapi): eso obligaría a empaquetar Chrome y a pagar un
// servidor siempre encendido para un microservicio de portafolio
// personal. En su lugar, la invoca cmd/shooter, un binario aparte que
// corre dentro de un workflow de GitHub Actions (que trae Chrome
// preinstalado y es gratis para repos públicos) cuando detecta que un
// proyecto con web desplegada tuvo un push nuevo. El PNG resultante se
// commitea a data/screenshots/{slug}.png, y el servidor en vivo lo sirve
// después vía CommittedScreenshotSource (committed.go), sin abrir Chrome
// nunca.
type ScreenshotSource struct {
	ChromePath string // opcional; si está vacío, se autodetecta
	Timeout    time.Duration
}

// NewScreenshotSource crea la fuente. chromePath puede ir vacío para
// autodetectar el binario de Chrome/Chromium instalado.
func NewScreenshotSource(chromePath string) *ScreenshotSource {
	return &ScreenshotSource{
		ChromePath: chromePath,
		Timeout:    30 * time.Second,
	}
}

func (s *ScreenshotSource) Name() string { return "screenshot" }

func (s *ScreenshotSource) Render(ctx context.Context, p catalog.Project, _ meta.Metadata) (Image, error) {
	if !p.HasLive() {
		return Image{}, ErrNotApplicable
	}

	// Si ChromePath viene vacío, chromedp autodetecta el binario de
	// Chrome/Chromium instalado (candidatos usuales: google-chrome,
	// chromium, chromium-browser, etc.) al momento de asignar el proceso.
	allocOpts := chromedp.DefaultExecAllocatorOptions[:]
	if s.ChromePath != "" {
		allocOpts = append(allocOpts, chromedp.ExecPath(s.ChromePath))
	}
	allocOpts = append(allocOpts,
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("hide-scrollbars", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, allocOpts...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	timeoutCtx, cancelTimeout := context.WithTimeout(taskCtx, s.Timeout)
	defer cancelTimeout()

	var buf []byte
	err := chromedp.Run(timeoutCtx,
		chromedp.EmulateViewport(1280, 800),
		chromedp.Navigate(p.Live),
		chromedp.WaitReady("body", chromedp.ByQuery),
		// Framer Motion / anime.js animan la carga inicial; sin esta
		// pausa la captura sale a mitad de la animación.
		chromedp.Sleep(2*time.Second),
		chromedp.CaptureScreenshot(&buf),
	)
	if err != nil {
		return Image{}, fmt.Errorf("preview: capturando %s: %w", p.Live, err)
	}

	return Image{Bytes: buf, ContentType: "image/png"}, nil
}
