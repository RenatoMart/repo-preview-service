package meta

import (
	"bytes"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// ImageCandidate es una imagen encontrada en un README, con su URL ya
// resuelta a una dirección absoluta.
type ImageCandidate struct {
	URL string
	Alt string
}

var mdRenderer = goldmark.New(goldmark.WithExtensions(extension.GFM))

// RenderHTML convierte el README (Markdown/GFM) a HTML para servirlo en
// la respuesta de /api/v1/projects/{slug}.
func RenderHTML(raw string) (string, error) {
	var buf bytes.Buffer
	if err := mdRenderer.Convert([]byte(raw), &buf); err != nil {
		return "", fmt.Errorf("meta: renderizando readme: %w", err)
	}
	return buf.String(), nil
}

// badgeHosts son dominios que sirven casi exclusivamente shields/badges,
// no capturas de la aplicación. Sin este filtro, la primera imagen de
// casi cualquier README sería un badge de "build: passing".
var badgeHosts = map[string]bool{
	"img.shields.io": true,
	"badgen.net":     true,
	"codecov.io":     true,
	"travis-ci.org":  true,
	"travis-ci.com":  true,
	"circleci.com":   true,
	"badge.fury.io":  true,
}

func isBadge(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if badgeHosts[strings.ToLower(u.Host)] {
		return true
	}
	return strings.Contains(strings.ToLower(u.Path), "badge")
}

var htmlImgTag = regexp.MustCompile(`(?is)<img\b[^>]*>`)
var htmlImgSrc = regexp.MustCompile(`(?is)\bsrc\s*=\s*["']([^"']+)["']`)
var htmlImgAlt = regexp.MustCompile(`(?is)\balt\s*=\s*["']([^"']*)["']`)

// ExtractImages recorre el README (parseándolo como Markdown y también
// buscando <img> HTML embebido, un patrón común para centrar imágenes)
// y devuelve las imágenes candidatas a preview, en el orden en que
// aparecen, con badges ya descartados y rutas relativas resueltas contra
// el repo y su rama por defecto.
func ExtractImages(raw, owner, repo, defaultBranch string) []ImageCandidate {
	if raw == "" {
		return nil
	}

	var out []ImageCandidate
	seen := make(map[string]bool)

	add := func(rawSrc, alt string) {
		if rawSrc == "" || isBadge(rawSrc) {
			return
		}
		resolved := resolveImageURL(rawSrc, owner, repo, defaultBranch)
		if seen[resolved] {
			return
		}
		seen[resolved] = true
		out = append(out, ImageCandidate{URL: resolved, Alt: alt})
	}

	source := []byte(raw)
	doc := mdRenderer.Parser().Parse(text.NewReader(source))
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if img, ok := n.(*ast.Image); ok {
			add(string(img.Destination), imageAltText(img, source))
		}
		return ast.WalkContinue, nil
	})

	for _, tag := range htmlImgTag.FindAllString(raw, -1) {
		srcMatch := htmlImgSrc.FindStringSubmatch(tag)
		if srcMatch == nil {
			continue
		}
		alt := ""
		if altMatch := htmlImgAlt.FindStringSubmatch(tag); altMatch != nil {
			alt = altMatch[1]
		}
		add(srcMatch[1], alt)
	}

	return out
}

func imageAltText(img *ast.Image, source []byte) string {
	var sb strings.Builder
	for c := img.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			sb.Write(t.Segment.Value(source))
		}
	}
	return sb.String()
}

func resolveImageURL(raw, owner, repo, defaultBranch string) string {
	if u, err := url.Parse(raw); err == nil && u.IsAbs() {
		return raw
	}
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	clean := strings.TrimPrefix(path.Clean("/"+raw), "/")
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, defaultBranch, clean)
}

// preferredKeywords son términos en el alt o la URL que sugieren que la
// imagen es una captura de la app en marcha, no un logo o diagrama.
var preferredKeywords = []string{"screenshot", "demo", "preview", "captura"}

// PickBestImage elige, entre las candidatas ya filtradas de badges, la
// que más probablemente sea una captura real de la aplicación: primero
// busca coincidencia por palabra clave en alt o URL, y si no hay
// ninguna, cae a la primera imagen en orden de aparición.
func PickBestImage(candidates []ImageCandidate) (ImageCandidate, bool) {
	if len(candidates) == 0 {
		return ImageCandidate{}, false
	}
	for _, kw := range preferredKeywords {
		for _, c := range candidates {
			hay := strings.ToLower(c.Alt + " " + c.URL)
			if strings.Contains(hay, kw) {
				return c, true
			}
		}
	}
	return candidates[0], true
}
