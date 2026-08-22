package preview

// langColors son los colores oficiales de GitHub Linguist para los
// lenguajes de tus repos, usados en la barra de proporción de la tarjeta
// generada (card.go). defaultLangColor cubre cualquier lenguaje no
// listado aquí.
var langColors = map[string]string{
	"Python":           "#3572A5",
	"TypeScript":       "#3178c6",
	"JavaScript":       "#f1e05a",
	"Go":               "#00ADD8",
	"Java":             "#b07219",
	"Jupyter Notebook": "#DA5B0B",
	"SWIG":             "#e2b5f9",
	"HTML":             "#e34c26",
	"CSS":              "#563d7c",
	"Shell":            "#89e051",
	"Dockerfile":       "#384d54",
	"C":                "#555555",
	"C++":              "#f34b7d",
	"C#":               "#178600",
	"Ruby":             "#701516",
	"PHP":              "#4F5D95",
	"Rust":             "#dea584",
	"Vue":              "#41b883",
	"Swift":            "#F05138",
	"Kotlin":           "#A97BFF",
	"Objective-C":      "#438eff",
	"Makefile":         "#427819",
	"PowerShell":       "#012456",
}

const defaultLangColor = "#8b8b8b"

func langColor(name string) string {
	if c, ok := langColors[name]; ok {
		return c
	}
	return defaultLangColor
}
