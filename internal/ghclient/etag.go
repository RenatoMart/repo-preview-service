package ghclient

import (
	"bytes"
	"io"
	"net/http"
	"sync"
)

// cachedResponse es lo que se guarda de una respuesta 200 para poder
// reproducirla cuando GitHub responde 304.
type cachedResponse struct {
	etag       string
	statusCode int
	header     http.Header
	body       []byte
}

// etagTransport es un http.RoundTripper que añade If-None-Match a las
// peticiones GET usando el ETag de la última respuesta 200 vista para esa
// URL, y traduce un 304 en la respuesta 200 cacheada.
//
// Esto es lo que hace que verificar "¿cambió algo?" no consuma cuota de la
// API de GitHub: una respuesta 304 no se descuenta del límite de
// peticiones por hora. go-github no sabe que este transport existe; para
// él, cada llamada recibe un 200 normal.
type etagTransport struct {
	base http.RoundTripper

	mu    sync.RWMutex
	cache map[string]cachedResponse // key: método+URL
}

func newETagTransport(base http.RoundTripper) *etagTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &etagTransport{
		base:  base,
		cache: make(map[string]cachedResponse),
	}
}

func (t *etagTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return t.base.RoundTrip(req)
	}

	key := req.URL.String()

	t.mu.RLock()
	cached, hasCache := t.cache[key]
	t.mu.RUnlock()

	if hasCache && cached.etag != "" {
		req = req.Clone(req.Context())
		req.Header.Set("If-None-Match", cached.etag)
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotModified && hasCache {
		// GitHub confirma que no hay cambios: descartamos el 304 (cuerpo
		// vacío) y devolvemos la respuesta 200 que teníamos guardada, como
		// si acabara de llegar. go-github la procesa con normalidad.
		_ = resp.Body.Close()
		return &http.Response{
			Status:        http.StatusText(cached.statusCode),
			StatusCode:    cached.statusCode,
			Proto:         resp.Proto,
			ProtoMajor:    resp.ProtoMajor,
			ProtoMinor:    resp.ProtoMinor,
			Header:        cached.header.Clone(),
			Body:          io.NopCloser(bytes.NewReader(cached.body)),
			ContentLength: int64(len(cached.body)),
			Request:       req,
		}, nil
	}

	if resp.StatusCode == http.StatusOK {
		etag := resp.Header.Get("ETag")
		if etag != "" {
			body, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr != nil {
				return nil, readErr
			}

			t.mu.Lock()
			t.cache[key] = cachedResponse{
				etag:       etag,
				statusCode: resp.StatusCode,
				header:     resp.Header.Clone(),
				body:       body,
			}
			t.mu.Unlock()

			resp.Body = io.NopCloser(bytes.NewReader(body))
			resp.ContentLength = int64(len(body))
		}
	}

	return resp, nil
}
