package preview

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"sort"
	"strings"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/meta"
)

// CardSource genera una tarjeta SVG con la identidad visual del proyecto:
// título, descripción, tags y la proporción real de lenguajes del repo.
// Es el paso final de la cascada (ver source.go) y por diseño nunca
// devuelve ErrNotApplicable: siempre hay algo que dibujar, así que el
// frontend nunca recibe un hueco roto, incluso para los proyectos que no
// tienen web desplegada ni imágenes en el README.
//
// Se eligió SVG (~2KB) sobre un PNG renderizado con fuentes embebidas:
// es nítido a cualquier densidad de pantalla y no añade dependencias de
// rasterizado.
type CardSource struct{}

// NewCardSource crea la fuente de tarjetas generadas.
func NewCardSource() *CardSource {
	return &CardSource{}
}

func (s *CardSource) Name() string { return "card" }

func (s *CardSource) Render(_ context.Context, p catalog.Project, m meta.Metadata) (Image, error) {
	svg, err := renderCard(p, m)
	if err != nil {
		return Image{}, fmt.Errorf("preview: generando tarjeta de %s: %w", p.Slug, err)
	}
	return Image{Bytes: []byte(svg), ContentType: "image/svg+xml"}, nil
}

const (
	cardWidth  = 1200
	cardHeight = 630
)

type langSlice struct {
	Name       string
	Percent    float64
	Color      string
	Width      float64 // en px, dentro de la barra
	X          float64 // offset en px, dentro de la barra
	LegendX    float64 // offset en px, dentro de la fila de leyenda
	LegendText string  // "Python 87%"
}

type tagPill struct {
	Text  string
	X     float64
	Width float64
}

type descLine struct {
	Text string
	Y    float64
}

type cardData struct {
	Width, Height int
	Accent        string
	Category      string
	Title         string
	TitleSize     int
	DescLines     []descLine
	Tags          []tagPill
	Langs         []langSlice
	BarX, BarY    float64
	BarWidth      float64
	LegendY       float64
}

var cardTemplate = template.Must(template.New("card").Parse(cardSVGTemplate))

func renderCard(p catalog.Project, m meta.Metadata) (string, error) {
	accent := p.Accent
	if accent == "" {
		accent = "#2563eb"
	}

	title := truncate(p.Title, 60)
	data := cardData{
		Width:     cardWidth,
		Height:    cardHeight,
		Accent:    accent,
		Category:  p.Category,
		Title:     title,
		TitleSize: titleFontSize(title),
		DescLines: layoutDescription(p.Description, 220, 34),
		Tags:      layoutTags(p.Tags, 64, float64(cardWidth)-128),
		BarX:      64,
		BarY:      520,
		BarWidth:  float64(cardWidth) - 128,
	}
	data.LegendY = data.BarY + 40
	data.Langs = layoutLanguages(m.Languages, data.BarWidth)

	var buf bytes.Buffer
	if err := cardTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// titleFontSize reduce el tamaño de letra del título según su largo, para
// que quepa en una sola línea dentro del ancho de la tarjeta.
func titleFontSize(title string) int {
	switch n := len([]rune(title)); {
	case n > 44:
		return 36
	case n > 30:
		return 44
	default:
		return 54
	}
}

func layoutDescription(desc string, startY, lineHeight float64) []descLine {
	var lines []descLine
	y := startY
	for _, l := range wrapText(desc, 54, 3) {
		lines = append(lines, descLine{Text: l, Y: y})
		y += lineHeight
	}
	return lines
}

// truncate corta s a lo sumo a maxRunes runas, añadiendo "…" si se cortó.
func truncate(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes-1]) + "…"
}

// wrapText parte s en líneas de a lo sumo maxChars caracteres (partiendo
// por palabras), limitado a maxLines; si sobra texto, la última línea se
// trunca con "…".
func wrapText(s string, maxChars, maxLines int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	line := ""
	i := 0
	for i < len(words) && len(lines) < maxLines {
		w := words[i]
		candidate := w
		if line != "" {
			candidate = line + " " + w
		}
		if len(candidate) > maxChars && line != "" {
			lines = append(lines, line)
			line = ""
			continue // reintenta la misma palabra en una línea nueva
		}
		line = candidate
		i++
	}
	if line != "" && len(lines) < maxLines {
		lines = append(lines, line)
	}

	if i < len(words) && len(lines) > 0 {
		// Sobró texto sin caber en maxLines: se marca con elipsis, sin
		// depender de que la última línea ya esté al límite de maxChars.
		last := []rune(lines[len(lines)-1])
		if len(last) > maxChars-1 {
			last = last[:maxChars-1]
		}
		lines[len(lines)-1] = string(last) + "…"
	}

	return lines
}

// layoutTags calcula el ancho de cada píldora a partir del largo del
// texto (estimación de 8.5px por carácter a 20px, más padding), y las va
// colocando en fila hasta agotar maxWidth; el resto de tags se descarta
// para no desbordar la tarjeta.
func layoutTags(tags []string, startX, maxWidth float64) []tagPill {
	const charWidth = 8.5
	const padding = 28
	const gap = 12

	var pills []tagPill
	x := startX
	for _, t := range tags {
		w := float64(len([]rune(t)))*charWidth + padding
		if x+w > startX+maxWidth {
			break
		}
		pills = append(pills, tagPill{Text: t, X: x, Width: w})
		x += w + gap
	}
	return pills
}

// layoutLanguages ordena los lenguajes por porcentaje descendente, se
// queda con los 5 más relevantes agrupando el resto en "Otros", y calcula
// la posición de cada segmento dentro de la barra.
func layoutLanguages(langs map[string]float64, barWidth float64) []langSlice {
	if len(langs) == 0 {
		return nil
	}

	type kv struct {
		name string
		pct  float64
	}
	all := make([]kv, 0, len(langs))
	for name, pct := range langs {
		all = append(all, kv{name, pct})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].pct > all[j].pct })

	const maxSlices = 5
	var top []kv
	var otherPct float64
	for i, e := range all {
		if i < maxSlices {
			top = append(top, e)
		} else {
			otherPct += e.pct
		}
	}
	if otherPct > 0.5 {
		top = append(top, kv{"Otros", otherPct})
	}

	slices := make([]langSlice, 0, len(top))
	x, legendX := 0.0, 0.0
	for _, e := range top {
		w := barWidth * e.pct / 100
		color := langColor(e.name)
		if e.name == "Otros" {
			color = defaultLangColor
		}
		legendText := fmt.Sprintf("%s %.0f%%", e.name, e.pct)

		slices = append(slices, langSlice{
			Name:       e.name,
			Percent:    e.pct,
			Color:      color,
			X:          x,
			Width:      w,
			LegendX:    legendX,
			LegendText: legendText,
		})
		x += w
		legendX += float64(len([]rune(legendText)))*7.5 + 36 // dot + texto + espacio al siguiente
	}
	return slices
}
