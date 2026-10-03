package preview

import (
	"context"
	"errors"
	"testing"

	"github.com/RenatoMart/repo-preview-service/internal/catalog"
	"github.com/RenatoMart/repo-preview-service/internal/meta"
)

type stubSource struct {
	name string
	err  error
}

func (s stubSource) Name() string { return s.name }
func (s stubSource) Render(context.Context, catalog.Project, meta.Metadata) (Image, error) {
	if s.err != nil {
		return Image{}, s.err
	}
	return Image{Bytes: []byte("x"), ContentType: "image/png"}, nil
}

func TestResolver_DegradedOnlyWhenHigherSourceFailedForReal(t *testing.T) {
	boom := errors.New("timeout")
	cases := []struct {
		name string
		srcs []Source
		want bool
	}{
		{"fallo real en la primera", []Source{stubSource{"screenshot", boom}, stubSource{"card", nil}}, true},
		{"no aplicable no degrada", []Source{stubSource{"screenshot", ErrNotApplicable}, stubSource{"card", nil}}, false},
		{"primera funciona", []Source{stubSource{"screenshot", nil}, stubSource{"card", nil}}, false},
	}
	for _, c := range cases {
		img, err := NewResolver(c.srcs...).Resolve(context.Background(), catalog.Project{Slug: "x"}, meta.Metadata{})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if img.Degraded != c.want {
			t.Errorf("%s: Degraded = %v, want %v", c.name, img.Degraded, c.want)
		}
	}
}
