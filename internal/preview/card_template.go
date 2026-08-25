package preview

// cardSVGTemplate genera la tarjeta de previsualización por defecto.
// Se renderiza con html/template (no text/template): aunque el
// documento resultante es SVG/XML, no HTML, las reglas de escapado de
// html/template para texto y atributos (&, <, >, ", ') son las mismas
// que necesita XML, así que sirve igual para escapar con seguridad
// título, descripción y tags que vienen de configs/projects.yaml.
const cardSVGTemplate = `<svg xmlns="http://www.w3.org/2000/svg" width="{{.Width}}" height="{{.Height}}" viewBox="0 0 {{.Width}} {{.Height}}">
  <title>{{.Title}}</title>
  {{if .Description}}<desc>{{.Description}}</desc>{{end}}
  <defs>
    <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0%" stop-color="#ffffff"/>
      <stop offset="100%" stop-color="#eef2f7"/>
    </linearGradient>
  </defs>

  <rect width="{{.Width}}" height="{{.Height}}" fill="url(#bg)"/>
  <rect x="0" y="0" width="{{.Width}}" height="6" fill="{{.Accent}}"/>

  <!-- Manchas decorativas con el color de acento: la tarjeta ya no
       repite título/descripción/tags como texto visible (eso lo
       muestra el frontend justo debajo de la imagen); esto le da
       identidad visual sin duplicar información. -->
  <circle cx="1020" cy="150" r="240" fill="{{.Accent}}" fill-opacity="0.07"/>
  <circle cx="140" cy="520" r="160" fill="{{.Accent}}" fill-opacity="0.05"/>

  {{if .Langs}}
  <rect x="{{.BarX}}" y="{{.BarY}}" width="{{.BarWidth}}" height="16" rx="8" fill="#e2e8f0"/>
  {{range .Langs}}<rect x="{{$.BarX}}" y="{{$.BarY}}" width="{{.Width}}" height="16"
        transform="translate({{.X}},0)" fill="{{.Color}}"/>
  {{end}}
  {{range .Langs}}<circle cx="{{$.BarX}}" cy="{{$.LegendY}}" r="5" transform="translate({{.LegendX}},0)" fill="{{.Color}}"/>
  <text x="{{$.BarX}}" y="{{$.LegendY}}" dx="14" dy="5" transform="translate({{.LegendX}},0)"
        font-family="system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif" font-size="16" fill="#334155">{{.LegendText}}</text>
  {{end}}
  {{end}}
</svg>
`
