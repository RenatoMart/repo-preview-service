package preview

// cardSVGTemplate genera la tarjeta de previsualización por defecto.
// Se renderiza con html/template (no text/template): aunque el
// documento resultante es SVG/XML, no HTML, las reglas de escapado de
// html/template para texto y atributos (&, <, >, ", ') son las mismas
// que necesita XML, así que sirve igual para escapar con seguridad
// título, descripción y tags que vienen de configs/projects.yaml.
const cardSVGTemplate = `<svg xmlns="http://www.w3.org/2000/svg" width="{{.Width}}" height="{{.Height}}" viewBox="0 0 {{.Width}} {{.Height}}">
  <defs>
    <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0%" stop-color="#0b1220"/>
      <stop offset="100%" stop-color="#111827"/>
    </linearGradient>
  </defs>

  <rect width="{{.Width}}" height="{{.Height}}" fill="url(#bg)"/>
  <rect x="0" y="0" width="{{.Width}}" height="6" fill="{{.Accent}}"/>

  <text x="64" y="76" font-family="system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
        font-size="20" font-weight="600" letter-spacing="2" fill="{{.Accent}}">{{.Category}}</text>

  <text x="64" y="150" font-family="system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
        font-size="{{.TitleSize}}" font-weight="700" fill="#f8fafc">{{.Title}}</text>

  {{range .DescLines}}<text x="64" y="{{.Y}}" font-family="system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
        font-size="24" fill="#94a3b8">{{.Text}}</text>
  {{end}}

  {{range .Tags}}<rect x="{{.X}}" y="320" width="{{.Width}}" height="40" rx="20"
        fill="{{$.Accent}}" fill-opacity="0.15" stroke="{{$.Accent}}" stroke-width="1.5"/>
  <text x="{{.X}}" y="346" dx="14" font-family="system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
        font-size="18" fill="{{$.Accent}}">{{.Text}}</text>
  {{end}}

  {{if .Langs}}
  <rect x="{{.BarX}}" y="{{.BarY}}" width="{{.BarWidth}}" height="14" rx="7" fill="#1e293b"/>
  {{range .Langs}}<rect x="{{$.BarX}}" y="{{$.BarY}}" width="{{.Width}}" height="14"
        transform="translate({{.X}},0)" fill="{{.Color}}"/>
  {{end}}
  {{range .Langs}}<circle cx="{{$.BarX}}" cy="{{$.LegendY}}" r="5" transform="translate({{.LegendX}},0)" fill="{{.Color}}"/>
  <text x="{{$.BarX}}" y="{{$.LegendY}}" dx="14" dy="5" transform="translate({{.LegendX}},0)"
        font-family="system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif" font-size="16" fill="#cbd5e1">{{.LegendText}}</text>
  {{end}}
  {{end}}
</svg>
`
