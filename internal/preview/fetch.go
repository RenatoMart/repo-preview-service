package preview

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// fetchImage descarga una imagen ya localizada por alguna de las fuentes
// (social preview, imagen del README) y la envuelve como Image.
func fetchImage(ctx context.Context, client *http.Client, url string) (Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Image{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return Image{}, fmt.Errorf("preview: descargando %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Image{}, fmt.Errorf("preview: descargando %s: status %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Image{}, err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}
	return Image{Bytes: body, ContentType: contentType}, nil
}
