# repo-preview-service

API en Go que sirve previsualizaciones (README, lenguajes, imagen) de los
proyectos de [personal-page](https://github.com/RenatoMart/personal-page),
para que el frontend en Next.js las muestre en las tarjetas del portafolio.

## Cómo funciona

**Metadatos (README, lenguajes, fecha de último push):** se traen de la
API de GitHub. Cada proyecto se refresca solo cuando `pushed_at` cambió
respecto a lo que ya se tenía guardado — con un transport HTTP propio que
usa ETags condicionales, verificar "¿cambió algo?" no gasta cuota de la
API (ver `internal/ghclient/etag.go`).

**Imagen de previsualización:** se resuelve con una cascada de fuentes,
probadas en orden hasta que una produzca algo (ver
`internal/preview/source.go`):

1. Captura real del sitio (si el proyecto tiene web desplegada) — la
   generó `cmd/shooter`, no este servidor (ver más abajo).
2. Social preview propia del repo (GitHub → Settings → Social preview).
3. La mejor imagen encontrada en el README (se filtran badges).
4. Tarjeta generada en SVG con título, descripción, tags y el porcentaje
   real de lenguajes — nunca falla, así que el frontend nunca recibe un
   hueco roto.

**Este servidor nunca abre un navegador.** Las capturas con chromedp las
toma `cmd/shooter`, corriendo dentro de un workflow de GitHub Actions
(gratis e ilimitado para repos públicos, y las máquinas de GitHub ya
traen Chrome instalado). El PNG resultante se commitea a
`data/screenshots/{slug}.png` y jsDelivr lo sirve desde ahí como CDN
gratuito. Eso es lo que permite desplegar `cmd/server` en un free tier
serverless (Cloud Run) sin empaquetar Chrome.

## Uso local

```bash
cp .env.example .env    # ajustar si hace falta; funciona con los valores por defecto
make run                # equivalente a: go run ./cmd/server

curl localhost:8080/healthz
curl localhost:8080/api/v1/projects | jq
curl localhost:8080/api/v1/projects/posture-corrector/preview -o preview.svg
```

Sin `GITHUB_TOKEN`, la API de GitHub limita a 60 peticiones/hora (igual
alcanza para probar). Con un PAT sin scopes en `.env`, sube a 5000/hora.

## Troubleshooting local

**"Ya subí una captura nueva pero el servidor sigue devolviendo la
tarjeta SVG":** el refresco automático (`internal/refresh/refresher.go`)
solo vuelve a resolver la cascada de un proyecto cuando `pushed_at` del
repo *de ese proyecto* cambió. Commitear una captura nueva a
`data/screenshots/` no toca el `pushed_at` de `ing-agroindustrial` ni de
ningún otro proyecto, así que si ya había una entrada cacheada en
`DATA_DIR/index.json`, se queda con el resultado viejo indefinidamente.
Para forzar que un slug se vuelva a resolver: borrar su entrada de
`index.json` (o el archivo entero) y reiniciar el servidor.

**"Reinicié el servidor pero sigue respondiendo lo mismo, o falla con
`address already in use`":** `go run` deja vivo un proceso hijo
(el binario ya compilado) aunque mates el proceso de `go run`. Buscar
quién tiene el puerto real con `lsof -i :8080` y matar ese PID, no el de
`go run`.

## Tomar capturas manualmente

```bash
make shoot                       # todos los proyectos con web desplegada
go run ./cmd/shooter -slug ing-agroindustrial -force
```

## Endpoints

| Método | Ruta | Qué hace |
|---|---|---|
| GET | `/healthz` | estado del servicio |
| GET | `/api/v1/projects` | catálogo completo, enriquecido |
| GET | `/api/v1/projects/{slug}` | detalle de un proyecto |
| GET | `/api/v1/projects/{slug}/preview` | imagen (SVG o PNG) |
| POST | `/api/v1/webhooks/github` | refresco de metadatos por push (HMAC) |
| POST | `/api/v1/admin/refresh[?slug=]` | refresco manual (token) |

## Añadir un proyecto

Editar `configs/projects.yaml`. No hace falta tocar código.

## Desplegar

1. **`cmd/server`** en Cloud Run (o cualquier plataforma con free tier
   serverless — no necesita Chrome):

   ```bash
   gcloud run deploy repo-preview --source . --region us-central1 \
     --allow-unauthenticated
   ```

   Configurar ahí las variables de `.env.example` (al menos
   `GITHUB_TOKEN`, `CORS_ALLOWED_ORIGINS`, y si se quiere webhook,
   `GITHUB_WEBHOOK_SECRET`).

2. **Capturas automáticas**: el workflow
   `.github/workflows/screenshots.yml` ya corre solo (cron cada 6h). Para
   que reaccione al instante a un push, copiar
   `docs/notify-preview.example.yml` como workflow dentro de cada repo
   con web desplegada (instrucciones en ese mismo archivo).

3. **Webhook de metadatos** (opcional): en cada repo del catálogo,
   Settings → Webhooks → Add webhook, apuntando a
   `https://tu-servicio/api/v1/webhooks/github`, evento `push`, con el
   mismo secreto que `GITHUB_WEBHOOK_SECRET`.
