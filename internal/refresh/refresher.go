// Package refresh implementa el ciclo de actualización de proyectos: por
// cada uno, primero pregunta lo barato (pushed_at, cacheado por ETag) y
// solo si cambió hace el trabajo caro (README, lenguajes, cascada de
// preview). Ver internal/ghclient/etag.go para por qué la verificación
// "¿cambió algo?" no consume cuota de la API de GitHub.
package refresh

import (
	"context"
	"log/slog"
	"time"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/meta"
	"github.com/RenatoMart/repo-preview-service/internal/preview"
	"github.com/RenatoMart/repo-preview-service/internal/store"
)

// queueSize acota cuántos refrescos pendientes se acumulan. Con 7
// proyectos es más que suficiente para absorber un ciclo del ticker más
// varios webhooks simultáneos.
const queueSize = 32

// Refresher orquesta el ciclo periódico de actualización y atiende tanto
// al ticker como a las peticiones puntuales del webhook, por la misma
// cola, para no duplicar trabajo sobre el mismo slug.
type Refresher struct {
	catalog  *catalog.Catalog
	meta     *meta.Service
	resolver *preview.Resolver
	store    *store.Store
	interval time.Duration

	queue chan string
}

// New crea un Refresher. Start debe llamarse para que empiece a operar.
func New(cat *catalog.Catalog, metaSvc *meta.Service, resolver *preview.Resolver, st *store.Store, interval time.Duration) *Refresher {
	return &Refresher{
		catalog:  cat,
		meta:     metaSvc,
		resolver: resolver,
		store:    st,
		interval: interval,
		queue:    make(chan string, queueSize),
	}
}

// Start lanza el worker y el ticker periódico. El primer refresco de
// todos los proyectos se encola en segundo plano, para que el servidor
// empiece a aceptar peticiones sin esperar a que termine. Se detiene
// cuando ctx se cancela.
func (r *Refresher) Start(ctx context.Context) {
	go r.worker(ctx)
	go r.enqueueAll()

	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.enqueueAll()
			}
		}
	}()
}

// enqueueAll encola todos los proyectos, pero primero los que todavía no
// tienen nada guardado: si la cuota de GitHub se agota a mitad del ciclo,
// lo que queda sin refrescar son proyectos que ya tienen datos, no los que
// aparecerían vacíos.
func (r *Refresher) enqueueAll() {
	var missing, known []string
	for _, p := range r.catalog.All() {
		if _, ok := r.store.Get(p.Slug); ok {
			known = append(known, p.Slug)
		} else {
			missing = append(missing, p.Slug)
		}
	}
	for _, slug := range append(missing, known...) {
		r.Enqueue(slug)
	}
}

// Enqueue pide refrescar un slug. La usan tanto el ticker como el
// webhook de GitHub. No bloquea: si la cola está llena, se descarta la
// petición (ya hay trabajo pendiente para ese mismo slug con alta
// probabilidad).
func (r *Refresher) Enqueue(slug string) {
	select {
	case r.queue <- slug:
	default:
		slog.Warn("refresh: cola llena, se descarta la petición", "slug", slug)
	}
}

func (r *Refresher) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case slug := <-r.queue:
			r.refreshOne(ctx, slug)
		}
	}
}

func (r *Refresher) refreshOne(ctx context.Context, slug string) {
	p, ok := r.catalog.BySlug(slug)
	if !ok {
		return
	}

	if p.NoAPI {
		r.refreshWithoutAPI(ctx, p)
		return
	}

	info, err := r.meta.RepoInfo(ctx, p)
	if err != nil {
		slog.ErrorContext(ctx, "refresh: repo info", "slug", slug, "err", err)
		r.storeWithoutGitHub(ctx, p)
		return
	}

	if existing, hasExisting := r.store.Get(slug); hasExisting &&
		!existing.Meta.PushedAt.IsZero() && existing.Meta.PushedAt.Equal(info.PushedAt) {
		if !existing.Preview.Degraded {
			// Nada cambió desde el último refresco. Gracias al ETag
			// transport, la llamada de arriba probablemente fue un 304
			// gratis, así que aquí no hace falta pedir README/lenguajes/
			// preview de nuevo.
			return
		}
		// El preview cayó a un fallback por un fallo transitorio (p. ej.
		// timeout del CDN). El repo no cambió, pero no hay que esperar a
		// un push: se reintenta solo la cascada de imagen, reutilizando la
		// metadata ya guardada (sin gastar cuota de GitHub).
		img, err := r.resolver.Resolve(ctx, p, existing.Meta)
		if err != nil {
			slog.ErrorContext(ctx, "refresh: reintentando preview degradado", "slug", slug, "err", err)
			return
		}
		existing.Preview = img
		existing.UpdatedAt = time.Now()
		if err := r.store.Set(slug, existing); err != nil {
			slog.ErrorContext(ctx, "refresh: guardando entry", "slug", slug, "err", err)
			return
		}
		slog.InfoContext(ctx, "refresh: preview reintentado", "slug", slug,
			"previewSource", img.Source, "degraded", img.Degraded)
		return
	}

	md, err := r.meta.Fetch(ctx, p, info)
	if err != nil {
		slog.ErrorContext(ctx, "refresh: fetch metadata", "slug", slug, "err", err)
		r.storeWithoutGitHub(ctx, p)
		return
	}

	img, err := r.resolver.Resolve(ctx, p, md)
	if err != nil {
		slog.ErrorContext(ctx, "refresh: resolviendo preview", "slug", slug, "err", err)
		return
	}

	entry := store.Entry{Project: p, Meta: md, Preview: img, UpdatedAt: time.Now()}
	if err := r.store.Set(slug, entry); err != nil {
		slog.ErrorContext(ctx, "refresh: guardando entry", "slug", slug, "err", err)
		return
	}

	slog.InfoContext(ctx, "refresh: proyecto actualizado", "slug", slug, "previewSource", img.Source)
}

// storeWithoutGitHub guarda lo que se pueda resolver sin la API de GitHub
// cuando esta falla (típicamente el límite de peticiones) y todavía no hay
// nada guardado para el proyecto: la captura real viene de jsDelivr y la
// tarjeta generada es local, así que el portafolio no se queda sin imágenes
// mientras dura el límite. La entrada queda con PushedAt vacío, lo que hace
// que el siguiente ciclo haga el refresco completo en cuanto GitHub responda.
// Si ya había una entrada (de un refresco anterior), no se toca.
func (r *Refresher) storeWithoutGitHub(ctx context.Context, p catalog.Project) {
	if _, ok := r.store.Get(p.Slug); ok {
		return
	}
	img, err := r.resolver.Resolve(ctx, p, meta.Metadata{})
	if err != nil {
		slog.ErrorContext(ctx, "refresh: preview sin GitHub", "slug", p.Slug, "err", err)
		return
	}
	entry := store.Entry{Project: p, Preview: img, UpdatedAt: time.Now()}
	if err := r.store.Set(p.Slug, entry); err != nil {
		slog.ErrorContext(ctx, "refresh: guardando entry", "slug", p.Slug, "err", err)
		return
	}
	slog.WarnContext(ctx, "refresh: preview provisional sin GitHub",
		"slug", p.Slug, "previewSource", img.Source)
}

// refreshWithoutAPI refresca un proyecto con no_api: nunca consulta la API
// de GitHub. La metadata sale solo del README crudo (gratis) y se reevalúa
// en cada ciclo, porque sin pushed_at no hay forma barata de saber si algo
// cambió; leer raw.githubusercontent.com no gasta cuota.
func (r *Refresher) refreshWithoutAPI(ctx context.Context, p catalog.Project) {
	md := r.meta.FetchFree(ctx, p)
	img, err := r.resolver.Resolve(ctx, p, md)
	if err != nil {
		slog.ErrorContext(ctx, "refresh: preview sin API", "slug", p.Slug, "err", err)
		return
	}
	entry := store.Entry{Project: p, Meta: md, Preview: img, UpdatedAt: time.Now()}
	if err := r.store.Set(p.Slug, entry); err != nil {
		slog.ErrorContext(ctx, "refresh: guardando entry", "slug", p.Slug, "err", err)
		return
	}
	slog.InfoContext(ctx, "refresh: proyecto actualizado sin API", "slug", p.Slug,
		"previewSource", img.Source)
}
