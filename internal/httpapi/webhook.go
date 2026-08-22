package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/refresh"
)

// maxWebhookBody limita el tamaño del payload aceptado, para no
// permitir que una petición maliciosa haga leer un cuerpo arbitrariamente
// grande antes de siquiera verificar la firma.
const maxWebhookBody = 1 << 20 // 1 MiB

// WebhookHandler recibe el evento "push" de GitHub para los proyectos
// del catálogo y encola un refresco de metadatos (README, lenguajes).
// No dispara capturas de pantalla: eso lo maneja un workflow de GitHub
// Actions aparte (ver cmd/shooter), no este servidor.
type WebhookHandler struct {
	secret    string
	catalog   *catalog.Catalog
	refresher *refresh.Refresher
}

// NewWebhookHandler crea el handler. Si secret está vacío, el endpoint
// rechaza todas las peticiones en vez de aceptar payloads sin verificar.
func NewWebhookHandler(secret string, cat *catalog.Catalog, r *refresh.Refresher) *WebhookHandler {
	return &WebhookHandler{secret: secret, catalog: cat, refresher: r}
}

type pushPayload struct {
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

// Handle responde POST /api/v1/webhooks/github.
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if h.secret == "" {
		writeError(w, http.StatusServiceUnavailable, "webhook no configurado")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody))
	if err != nil {
		writeError(w, http.StatusBadRequest, "no se pudo leer el cuerpo")
		return
	}

	if !validSignature(h.secret, body, r.Header.Get("X-Hub-Signature-256")) {
		writeError(w, http.StatusUnauthorized, "firma inválida")
		return
	}

	if r.Header.Get("X-GitHub-Event") != "push" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignorado (no es push)"})
		return
	}

	var payload pushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "payload inválido")
		return
	}

	p, ok := h.catalog.ByRepo(payload.Repository.FullName)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]string{"status": "repo no está en el catálogo"})
		return
	}

	h.refresher.Enqueue(p.Slug)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "refresco encolado", "slug": p.Slug})
}

// validSignature verifica la cabecera X-Hub-Signature-256 con
// hmac.Equal (comparación en tiempo constante) en vez de comparar los
// hex directamente, para no filtrar el secreto por temporización.
func validSignature(secret string, body []byte, header string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	got := strings.TrimPrefix(header, prefix)
	return hmac.Equal([]byte(expected), []byte(got))
}
