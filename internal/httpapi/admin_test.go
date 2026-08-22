package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RenatoMart/repo-preview-service/internal/refresh"
)

func TestAdminHandler_Refresh_Authorization(t *testing.T) {
	cat := testCatalog(t)
	ref := refresh.New(cat, nil, nil, nil, time.Minute)
	h := NewAdminHandler("s3cr3t-token", cat, ref)

	call := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/refresh", nil)
		if token != "" {
			req.Header.Set("X-Admin-Token", token)
		}
		rec := httptest.NewRecorder()
		h.Refresh(rec, req)
		return rec
	}

	if rec := call(""); rec.Code != http.StatusUnauthorized {
		t.Errorf("sin token: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if rec := call("token-equivocado"); rec.Code != http.StatusUnauthorized {
		t.Errorf("token equivocado: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if rec := call("s3cr3t-token"); rec.Code != http.StatusAccepted {
		t.Errorf("token correcto: status = %d, want %d", rec.Code, http.StatusAccepted)
	}
}

func TestAdminHandler_Refresh_NoTokenConfiguredRejectsAll(t *testing.T) {
	cat := testCatalog(t)
	ref := refresh.New(cat, nil, nil, nil, time.Minute)
	h := NewAdminHandler("", cat, ref)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/refresh", nil)
	rec := httptest.NewRecorder()
	h.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAdminHandler_Refresh_UnknownSlug(t *testing.T) {
	cat := testCatalog(t)
	ref := refresh.New(cat, nil, nil, nil, time.Minute)
	h := NewAdminHandler("s3cr3t-token", cat, ref)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/refresh?slug=no-existe", nil)
	req.Header.Set("X-Admin-Token", "s3cr3t-token")
	rec := httptest.NewRecorder()
	h.Refresh(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
