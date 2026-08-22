package preview

import (
	"context"
	"strings"
	"testing"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/meta"
)

func TestTruncate(t *testing.T) {
	if got := truncate("hola", 10); got != "hola" {
		t.Errorf("truncate corto = %q, want sin cambios", got)
	}
	if got := truncate("holamundolargo", 8); got != "holamun…" {
		t.Errorf("truncate largo = %q, want %q", got, "holamun…")
	}
}

func TestWrapText_FitsWithoutTruncating(t *testing.T) {
	got := wrapText("una linea corta", 54, 3)
	if len(got) != 1 || got[0] != "una linea corta" {
		t.Errorf("wrapText() = %q, want una sola línea sin cambios", got)
	}
}

func TestWrapText_TruncatesWhenOverflowsMaxLines(t *testing.T) {
	long := strings.Repeat("palabra ", 40) // MUY largo, no cabe en 3 líneas de 20 chars
	got := wrapText(long, 20, 3)
	if len(got) != 3 {
		t.Fatalf("wrapText() devolvió %d líneas, want 3", len(got))
	}
	last := got[len(got)-1]
	if !strings.HasSuffix(last, "…") {
		t.Errorf("última línea = %q, want que termine en elipsis", last)
	}
}

func TestWrapText_Empty(t *testing.T) {
	if got := wrapText("", 20, 3); got != nil {
		t.Errorf("wrapText(\"\") = %v, want nil", got)
	}
}

func TestLayoutTags_DropsOverflow(t *testing.T) {
	tags := []string{"Python", "MediaPipe", "Machine Learning", "Computer Vision", "OpenCV", "Extra"}
	pills := layoutTags(tags, 64, 300) // ancho chico a propósito
	if len(pills) == 0 || len(pills) >= len(tags) {
		t.Fatalf("layoutTags() = %d píldoras de %d tags, want que se descarten las que no caben", len(pills), len(tags))
	}
	for i := 1; i < len(pills); i++ {
		if pills[i].X <= pills[i-1].X {
			t.Errorf("pills[%d].X = %v, want mayor que pills[%d].X = %v", i, pills[i].X, i-1, pills[i-1].X)
		}
	}
}

func TestLayoutLanguages_GroupsIntoOtros(t *testing.T) {
	langs := map[string]float64{
		"Python": 40, "TypeScript": 20, "Go": 15, "Java": 10, "C++": 8, "Ruby": 5, "PHP": 2,
	}
	slices := layoutLanguages(langs, 1000)

	var hasOtros bool
	var total float64
	for _, s := range slices {
		total += s.Percent
		if s.Name == "Otros" {
			hasOtros = true
		}
	}
	if !hasOtros {
		t.Error("layoutLanguages() no agrupó el excedente en 'Otros'")
	}
	if total < 99 || total > 101 {
		t.Errorf("suma de porcentajes = %.2f, want ~100", total)
	}
}

func TestLayoutLanguages_Empty(t *testing.T) {
	if got := layoutLanguages(nil, 1000); got != nil {
		t.Errorf("layoutLanguages(nil) = %v, want nil", got)
	}
}

func TestRenderCard_EscapesUntrustedLookingChars(t *testing.T) {
	p := catalog.Project{
		Slug:        "test",
		Title:       "A & B <script>",
		Category:    "Trabajos Web",
		Accent:      "#2563eb",
		Tags:        []string{"Go", "SVG"},
		Description: "Descripción con \"comillas\" y <tags>",
	}
	svg, err := renderCard(p, meta.Metadata{Languages: map[string]float64{"Go": 100}})
	if err != nil {
		t.Fatalf("renderCard: %v", err)
	}
	if strings.Contains(svg, "<script>") {
		t.Error("renderCard() no escapó '<script>' del título, riesgo de XSS/XML inválido")
	}
	if !strings.Contains(svg, "&amp;") {
		t.Error("renderCard() no escapó '&' del título")
	}
	if !strings.Contains(svg, "<svg") || !strings.Contains(svg, "</svg>") {
		t.Error("renderCard() no produjo un documento SVG bien formado")
	}
}

func TestCardSource_NeverReturnsNotApplicable(t *testing.T) {
	s := NewCardSource()
	p := catalog.Project{Slug: "x", Title: "X", Accent: "#000"}
	img, err := s.Render(context.Background(), p, meta.Metadata{})
	if err != nil {
		t.Fatalf("CardSource.Render() con metadata vacío: %v", err)
	}
	if img.ContentType != "image/svg+xml" || len(img.Bytes) == 0 {
		t.Errorf("CardSource.Render() = %+v, want SVG no vacío", img)
	}
}
