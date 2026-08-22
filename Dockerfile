# Imagen del servidor HTTP en vivo (cmd/server). Nunca ejecuta Chrome —
# las capturas de pantalla se generan aparte, con cmd/shooter, dentro de
# un workflow de GitHub Actions (ver .github/workflows/screenshots.yml) —
# así que esta imagen es un binario Go simple, sin navegador empaquetado.
# Eso es lo que la hace viable en un free tier serverless como Cloud Run:
# es liviana y arranca en milisegundos.

FROM golang:1.27-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=build /out/server ./server
COPY configs ./configs

# Cloud Run inyecta PORT en runtime; 8080 es el valor por defecto local.
# DATA_DIR va a /tmp: es el único directorio garantizado escribible para
# el usuario "nonroot" de la imagen distroless, y de todos modos el
# caché en un contenedor serverless no sobrevive entre instancias — el
# primer refresco en cada arranque lo repuebla.
ENV PORT=8080
ENV DATA_DIR=/tmp/data
EXPOSE 8080

ENTRYPOINT ["./server"]
