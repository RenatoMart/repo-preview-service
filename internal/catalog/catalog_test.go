package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "projects.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("escribiendo yaml temporal: %v", err)
	}
	return path
}

func TestLoad_Valid(t *testing.T) {
	path := writeTemp(t, `
- slug: foo
  repo: RenatoMart/foo
  title: Foo
  category: Trabajos Web
  live: "https://foo.vercel.app"
- slug: bar
  repo: RenatoMart/bar
  title: Bar
  category: Proyectos Personales
  live: ""
`)

	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", c.Len())
	}

	foo, ok := c.BySlug("foo")
	if !ok {
		t.Fatal("BySlug(foo) no encontrado")
	}
	if !foo.HasLive() {
		t.Error("foo.HasLive() = false, want true")
	}
	if foo.Owner() != "RenatoMart" || foo.Name() != "foo" {
		t.Errorf("Owner/Name = %q/%q, want RenatoMart/foo", foo.Owner(), foo.Name())
	}

	bar, ok := c.ByRepo("renatomart/bar") // case-insensitive, como llega del webhook
	if !ok {
		t.Fatal("ByRepo(renatomart/bar) no encontrado")
	}
	if bar.HasLive() {
		t.Error("bar.HasLive() = true, want false")
	}
}

func TestLoad_DuplicateSlug(t *testing.T) {
	path := writeTemp(t, `
- slug: foo
  repo: RenatoMart/foo
  title: Foo
- slug: foo
  repo: RenatoMart/foo2
  title: Foo2
`)
	if _, err := Load(path); err == nil {
		t.Fatal("Load: esperaba error por slug duplicado, obtuvo nil")
	}
}

func TestLoad_InvalidRepoFormat(t *testing.T) {
	path := writeTemp(t, `
- slug: foo
  repo: not-a-valid-repo
  title: Foo
`)
	if _, err := Load(path); err == nil {
		t.Fatal("Load: esperaba error por formato de repo inválido, obtuvo nil")
	}
}

func TestLoad_LiveMustBeHTTPS(t *testing.T) {
	path := writeTemp(t, `
- slug: foo
  repo: RenatoMart/foo
  title: Foo
  live: "http://inseguro.com"
`)
	if _, err := Load(path); err == nil {
		t.Fatal("Load: esperaba error por live no-https, obtuvo nil")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	if _, err := Load("/no/existe/projects.yaml"); err == nil {
		t.Fatal("Load: esperaba error por archivo inexistente, obtuvo nil")
	}
}

func TestLoad_Empty(t *testing.T) {
	path := writeTemp(t, `[]`)
	if _, err := Load(path); err == nil {
		t.Fatal("Load: esperaba error por catálogo vacío, obtuvo nil")
	}
}
