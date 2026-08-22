package ghclient

import (
	"errors"
	"io"
	"regexp"
)

// maxOGImageScan limita cuánto HTML se lee al buscar el meta tag og:image.
// El tag está siempre en <head>, así que no hace falta descargar la
// página entera.
const maxOGImageScan = 256 * 1024

var ogImagePattern = regexp.MustCompile(
	`<meta[^>]+property=["']og:image["'][^>]+content=["']([^"']+)["']|` +
		`<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:image["']`,
)

// ErrOGImageNotFound se devuelve cuando la página no tiene meta tag og:image.
var ErrOGImageNotFound = errors.New("ghclient: og:image no encontrado")

func extractOGImage(r io.Reader) (string, error) {
	body, err := io.ReadAll(io.LimitReader(r, maxOGImageScan))
	if err != nil {
		return "", err
	}

	m := ogImagePattern.FindSubmatch(body)
	if m == nil {
		return "", ErrOGImageNotFound
	}
	if len(m[1]) > 0 {
		return string(m[1]), nil
	}
	return string(m[2]), nil
}
