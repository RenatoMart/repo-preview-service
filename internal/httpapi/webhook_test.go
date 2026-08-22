package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/refresh"
)

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestValidSignature(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	secret := "s3cr3t"

	if !validSignature(secret, body, sign(secret, body)) {
		t.Error("validSignature() = false para una firma correcta, want true")
	}
	if validSignature(secret, body, sign("otro-secreto", body)) {
		t.Error("validSignature() = true con el secreto equivocado, want false")
	}
	if validSignature(secret, body, "no-tiene-prefijo") {
		t.Error("validSignature() = true sin el prefijo sha256=, want false")
	}
	if validSignature(secret, []byte("cuerpo distinto"), sign(secret, body)) {
		t.Error("validSignature() = true con un cuerpo distinto al firmado, want false")
	}
}

func testCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	path := filepath.Join(t.TempDir(), "projects.yaml")
	content := `
- slug: foo
  repo: RenatoMart/foo
  title: Foo
  category: Trabajos Web
  live: "https://foo.vercel.app"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("escribiendo catálogo temporal: %v", err)
	}
	c, err := catalog.Load(path)
	if err != nil {
		t.Fatalf("catalog.Load: %v", err)
	}
	return c
}

func TestWebhookHandler_Handle(t *testing.T) {
	cat := testCatalog(t)
	ref := refresh.New(cat, nil, nil, nil, time.Minute) // Start() nunca se llama: Enqueue solo escribe en un canal con buffer

	const secret = "s3cr3t"
	h := NewWebhookHandler(secret, cat, ref)

	post := func(body []byte, sig, event string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", strings.NewReader(string(body)))
		if sig != "" {
			req.Header.Set("X-Hub-Signature-256", sig)
		}
		if event != "" {
			req.Header.Set("X-GitHub-Event", event)
		}
		rec := httptest.NewRecorder()
		h.Handle(rec, req)
		return rec
	}

	t.Run("firma inválida", func(t *testing.T) {
		body := []byte(`{"repository":{"full_name":"RenatoMart/foo"}}`)
		rec := post(body, "sha256=deadbeef", "push")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("evento distinto de push se ignora", func(t *testing.T) {
		body := []byte(`{"repository":{"full_name":"RenatoMart/foo"}}`)
		rec := post(body, sign(secret, body), "star")
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("repo fuera del catálogo se acepta sin encolar", func(t *testing.T) {
		body := []byte(`{"repository":{"full_name":"otro/repo-no-listado"}}`)
		rec := post(body, sign(secret, body), "push")
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("push válido de un repo del catálogo encola el refresco", func(t *testing.T) {
		body := []byte(`{"repository":{"full_name":"RenatoMart/foo"}}`)
		rec := post(body, sign(secret, body), "push")
		if rec.Code != http.StatusAccepted {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusAccepted)
		}
	})

	t.Run("sin secreto configurado, rechaza todo", func(t *testing.T) {
		h := NewWebhookHandler("", cat, ref)
		body := []byte(`{"repository":{"full_name":"RenatoMart/foo"}}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", strings.NewReader(string(body)))
		rec := httptest.NewRecorder()
		h.Handle(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})
}
