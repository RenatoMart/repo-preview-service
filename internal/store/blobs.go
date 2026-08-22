package store

import (
	"os"
	"path/filepath"
)

// extensionFor mapea un content-type a la extensión con la que se guarda
// el blob en disco. Solo importa para que el archivo sea legible al
// inspeccionar data/blobs/ a mano; el índice es quien decide qué
// content-type servir.
func extensionFor(contentType string) string {
	switch contentType {
	case "image/svg+xml":
		return ".svg"
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}

// writeFileAtomic escribe a un archivo temporal y lo renombra sobre el
// destino final. Un reinicio a mitad de escritura deja el archivo
// anterior intacto en vez de un archivo a medio truncar.
func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
