package meta

import "testing"

func TestExtractImages_FiltersBadgesAndResolvesRelative(t *testing.T) {
	raw := `# Mi Proyecto

![build](https://img.shields.io/github/workflow/status/foo/bar/CI)
![coverage](https://codecov.io/gh/foo/bar/badge.svg)

Una captura de la app:

![demo](docs/demo.png)

<p align="center"><img src="./assets/otra.png" alt="pantalla principal"></p>
`

	got := ExtractImages(raw, "RenatoMart", "posture-corrector", "main")

	if len(got) != 2 {
		t.Fatalf("ExtractImages() devolvió %d imágenes, want 2 (badges deben filtrarse): %+v", len(got), got)
	}

	want0 := "https://raw.githubusercontent.com/RenatoMart/posture-corrector/main/docs/demo.png"
	if got[0].URL != want0 {
		t.Errorf("got[0].URL = %q, want %q", got[0].URL, want0)
	}

	want1 := "https://raw.githubusercontent.com/RenatoMart/posture-corrector/main/assets/otra.png"
	if got[1].URL != want1 {
		t.Errorf("got[1].URL = %q, want %q", got[1].URL, want1)
	}
}

func TestExtractImages_AbsoluteURLKeptAsIs(t *testing.T) {
	raw := `![demo](https://cdn.example.com/shot.png)`
	got := ExtractImages(raw, "owner", "repo", "main")
	if len(got) != 1 || got[0].URL != "https://cdn.example.com/shot.png" {
		t.Fatalf("ExtractImages() = %+v, want URL absoluta sin modificar", got)
	}
}

func TestExtractImages_Deduplicates(t *testing.T) {
	raw := `![a](docs/x.png)
![b](./docs/x.png)`
	got := ExtractImages(raw, "owner", "repo", "main")
	if len(got) != 1 {
		t.Fatalf("ExtractImages() devolvió %d imágenes, want 1 (deben deduplicarse tras resolver la ruta)", len(got))
	}
}

func TestPickBestImage_PrefersKeywordMatch(t *testing.T) {
	candidates := []ImageCandidate{
		{URL: "https://raw.githubusercontent.com/o/r/main/logo.png", Alt: "logo"},
		{URL: "https://raw.githubusercontent.com/o/r/main/screenshot.png", Alt: ""},
	}
	best, ok := PickBestImage(candidates)
	if !ok {
		t.Fatal("PickBestImage() ok = false, want true")
	}
	if best.URL != candidates[1].URL {
		t.Errorf("PickBestImage() = %q, want la que contiene 'screenshot' en la URL", best.URL)
	}
}

func TestPickBestImage_FallsBackToFirst(t *testing.T) {
	candidates := []ImageCandidate{
		{URL: "https://raw.githubusercontent.com/o/r/main/a.png"},
		{URL: "https://raw.githubusercontent.com/o/r/main/b.png"},
	}
	best, ok := PickBestImage(candidates)
	if !ok || best.URL != candidates[0].URL {
		t.Errorf("PickBestImage() = %+v, ok=%v, want la primera en orden de aparición", best, ok)
	}
}

func TestPickBestImage_Empty(t *testing.T) {
	if _, ok := PickBestImage(nil); ok {
		t.Error("PickBestImage(nil) ok = true, want false")
	}
}

func TestRenderHTML(t *testing.T) {
	html, err := RenderHTML("# Título\n\nTexto con **negrita**.")
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if html == "" {
		t.Error("RenderHTML() devolvió vacío")
	}
}
