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

  <!-- La tarjeta no repite título/descripción/tags como texto visible
       (eso lo muestra el frontend justo debajo de la imagen); en su
       lugar construye identidad visual con la marca del proyecto y la
       proporción real de lenguajes del repo (anillo + leyenda). -->
  <circle cx="1060" cy="600" r="200" fill="{{.Accent}}" fill-opacity="0.05"/>
  <circle cx="60" cy="20" r="110" fill="{{.Accent}}" fill-opacity="0.05"/>

  <!-- Marca: distintivo "de código" con el color de acento del proyecto. -->
  <rect x="64" y="64" width="92" height="92" rx="22" fill="{{.Accent}}" fill-opacity="0.12" stroke="{{.Accent}}" stroke-opacity="0.35" stroke-width="1.5"/>
  <text x="110" y="123" text-anchor="middle" font-family="'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, Consolas, monospace" font-size="34" font-weight="600" fill="{{.Accent}}">&lt;/&gt;</text>

  {{if .Langs}}
  <circle cx="{{.RingCX}}" cy="{{.RingCY}}" r="{{.RingR}}" fill="none" stroke="#e2e8f0" stroke-width="{{.RingStroke}}"/>
  <g transform="rotate(-90 {{.RingCX}} {{.RingCY}})">
    {{range .Langs}}<circle cx="{{$.RingCX}}" cy="{{$.RingCY}}" r="{{$.RingR}}" fill="none" stroke="{{.Color}}"
        stroke-width="{{$.RingStroke}}" stroke-dasharray="{{.ArcDashArray}}" stroke-dashoffset="{{.ArcOffset}}"/>
    {{end}}
  </g>
  <text x="{{.RingCX}}" y="{{.RingCY}}" dy="-4" text-anchor="middle" font-family="system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif" font-size="26" font-weight="700" fill="#1e293b">{{.TopLangName}}</text>
  <text x="{{.RingCX}}" y="{{.RingCY}}" dy="24" text-anchor="middle" font-family="system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif" font-size="15" fill="#64748b">{{.TopLangPercent}}</text>

  {{range .Legend}}<circle cx="{{.DotCX}}" cy="{{.DotCY}}" r="5" fill="{{.Color}}"/>
  <text x="{{.TextX}}" y="{{.TextY}}" font-family="system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif" font-size="17" font-weight="500" fill="#334155">{{.Text}}</text>
  {{end}}
  {{else}}
  <!-- Sin datos de lenguaje (repo vacío o metadata no disponible): una
       marca fantasma centrada evita que la tarjeta quede casi en blanco. -->
  <text x="{{.RingCX}}" y="{{.RingCY}}" dy="50" text-anchor="middle" font-family="'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, Consolas, monospace" font-size="150" font-weight="600" fill="{{.Accent}}" fill-opacity="0.06">&lt;/&gt;</text>
  {{end}}
</svg>
`
