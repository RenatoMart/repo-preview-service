// Package httpapi expone la API HTTP del servicio: catálogo enriquecido,
// imagen de preview, y los dos endpoints que disparan un refresco
// (webhook de GitHub y refresco manual protegido por token).
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("httpapi: escribiendo respuesta JSON", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
