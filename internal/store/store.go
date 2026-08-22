// Package store mantiene, en memoria y persistido a disco, el resultado
// ya procesado de cada proyecto (metadatos + imagen de preview elegida
// por la cascada), para que las peticiones HTTP nunca tengan que esperar
// a una llamada a GitHub: eso lo hace por separado internal/refresh.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/meta"
	"github.com/RenatoMart/repo-preview-service/internal/preview"
)

// Entry es todo lo que se sabe de un proyecto tras el último refresco.
type Entry struct {
	Project   catalog.Project
	Meta      meta.Metadata
	Preview   preview.Image
	UpdatedAt time.Time
}

// indexRecord es lo que se persiste en data/index.json: todo Entry menos
// los bytes de la imagen, que van aparte en data/blobs/ para no convertir
// el índice en un archivo gigante.
type indexRecord struct {
	Project            catalog.Project `json:"project"`
	Meta               meta.Metadata   `json:"meta"`
	PreviewSource      string          `json:"previewSource"`
	PreviewContentType string          `json:"previewContentType"`
	UpdatedAt          time.Time       `json:"updatedAt"`
}

// Store es el índice en memoria, respaldado por archivos en dataDir.
type Store struct {
	dataDir string

	mu      sync.RWMutex
	entries map[string]Entry
}

// New crea un Store que persiste bajo dataDir. Hay que llamar a Load
// para recuperar el estado de un arranque anterior.
func New(dataDir string) *Store {
	return &Store{dataDir: dataDir, entries: make(map[string]Entry)}
}

func (s *Store) indexPath() string {
	return filepath.Join(s.dataDir, "index.json")
}

func (s *Store) blobPath(slug, contentType string) string {
	return filepath.Join(s.dataDir, "blobs", slug+extensionFor(contentType))
}

// Load recupera el estado guardado en un arranque anterior. Si no existe
// índice previo (primer arranque), no es un error: el Store simplemente
// queda vacío hasta el primer refresco.
func (s *Store) Load() error {
	raw, err := os.ReadFile(s.indexPath())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("store: leyendo índice: %w", err)
	}

	var records map[string]indexRecord
	if err := json.Unmarshal(raw, &records); err != nil {
		return fmt.Errorf("store: parseando índice: %w", err)
	}

	entries := make(map[string]Entry, len(records))
	for slug, r := range records {
		blob, err := os.ReadFile(s.blobPath(slug, r.PreviewContentType))
		if err != nil {
			continue // blob perdido/corrupto: se regenerará en el próximo refresco
		}
		entries[slug] = Entry{
			Project: r.Project,
			Meta:    r.Meta,
			Preview: preview.Image{
				Bytes:       blob,
				ContentType: r.PreviewContentType,
				Source:      r.PreviewSource,
			},
			UpdatedAt: r.UpdatedAt,
		}
	}

	s.mu.Lock()
	s.entries = entries
	s.mu.Unlock()
	return nil
}

// Get busca la entrada de un slug.
func (s *Store) Get(slug string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[slug]
	return e, ok
}

// All devuelve todas las entradas guardadas.
func (s *Store) All() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		out = append(out, e)
	}
	return out
}

// Set guarda una entrada en memoria y la persiste a disco (blob de la
// imagen + índice), para que un reinicio no tenga que rehacer el trabajo
// ni gastar cuota de GitHub de nuevo.
func (s *Store) Set(slug string, e Entry) error {
	blobPath := s.blobPath(slug, e.Preview.ContentType)
	if err := writeFileAtomic(blobPath, e.Preview.Bytes); err != nil {
		return fmt.Errorf("store: guardando blob de %s: %w", slug, err)
	}

	s.mu.Lock()
	s.entries[slug] = e
	records := s.snapshotRecordsLocked()
	s.mu.Unlock()

	raw, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("store: serializando índice: %w", err)
	}
	if err := writeFileAtomic(s.indexPath(), raw); err != nil {
		return fmt.Errorf("store: guardando índice: %w", err)
	}
	return nil
}

func (s *Store) snapshotRecordsLocked() map[string]indexRecord {
	records := make(map[string]indexRecord, len(s.entries))
	for slug, e := range s.entries {
		records[slug] = indexRecord{
			Project:            e.Project,
			Meta:               e.Meta,
			PreviewSource:      e.Preview.Source,
			PreviewContentType: e.Preview.ContentType,
			UpdatedAt:          e.UpdatedAt,
		}
	}
	return records
}
