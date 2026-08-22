package store

import (
	"testing"
	"time"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/meta"
	"github.com/RenatoMart/repo-preview-service/internal/preview"
)

func TestStore_SetAndGet(t *testing.T) {
	s := New(t.TempDir())

	entry := Entry{
		Project: catalog.Project{Slug: "foo", Title: "Foo"},
		Meta:    meta.Metadata{PushedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		Preview: preview.Image{Bytes: []byte("<svg></svg>"), ContentType: "image/svg+xml", Source: "card"},
	}
	if err := s.Set("foo", entry); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, ok := s.Get("foo")
	if !ok {
		t.Fatal("Get(foo) no encontrado justo después de Set")
	}
	if string(got.Preview.Bytes) != "<svg></svg>" {
		t.Errorf("Preview.Bytes = %q, want %q", got.Preview.Bytes, "<svg></svg>")
	}
}

func TestStore_LoadRecoversAfterRestart(t *testing.T) {
	dir := t.TempDir()

	s1 := New(dir)
	entry := Entry{
		Project:   catalog.Project{Slug: "bar", Title: "Bar"},
		Meta:      meta.Metadata{PushedAt: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC), Languages: map[string]float64{"Go": 100}},
		Preview:   preview.Image{Bytes: []byte{0x89, 0x50, 0x4e, 0x47}, ContentType: "image/png", Source: "screenshot"},
		UpdatedAt: time.Now(),
	}
	if err := s1.Set("bar", entry); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Simula un reinicio: un Store nuevo sobre el mismo directorio.
	s2 := New(dir)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	got, ok := s2.Get("bar")
	if !ok {
		t.Fatal("Get(bar) no encontrado tras Load, la persistencia no sobrevivió al reinicio")
	}
	if got.Preview.ContentType != "image/png" || len(got.Preview.Bytes) != 4 {
		t.Errorf("Preview tras reinicio = %+v, want el PNG guardado", got.Preview)
	}
	if got.Meta.Languages["Go"] != 100 {
		t.Errorf("Meta.Languages tras reinicio = %+v, want Go:100", got.Meta.Languages)
	}
}

func TestStore_Load_NoPreviousIndexIsNotAnError(t *testing.T) {
	s := New(t.TempDir())
	if err := s.Load(); err != nil {
		t.Fatalf("Load() en un directorio vacío: %v, want nil (primer arranque)", err)
	}
	if len(s.All()) != 0 {
		t.Errorf("All() = %d entradas, want 0", len(s.All()))
	}
}
