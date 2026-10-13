package pinned

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/meop/golie/fetch"
)

// Read downloads the archive at url and returns the Asset that pins it: its
// sha256, and that of each of files inside it. It trusts what it is given —
// it is for writing a pin down, with cmd/pin, not for checking one.
func Read(ctx context.Context, url string, files []string) (Asset, error) {
	work, err := os.MkdirTemp("", "pin-*")
	if err != nil {
		return Asset{}, err
	}
	defer func() { _ = os.RemoveAll(work) }()

	archive := filepath.Join(work, filepath.Base(url))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Asset{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Asset{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return Asset{}, fmt.Errorf("%s: %s", url, resp.Status)
	}
	out, err := os.Create(archive)
	if err != nil {
		return Asset{}, err
	}
	_, copyErr := io.Copy(out, resp.Body)
	if err := out.Close(); copyErr != nil || err != nil {
		return Asset{}, fmt.Errorf("%s: download failed", url)
	}

	asset := Asset{URL: url, Files: map[string]string{}}
	if asset.SHA256, err = fileSHA256(archive); err != nil {
		return Asset{}, err
	}
	unpacked := filepath.Join(work, "unpacked")
	if err := fetch.Extract(archive, unpacked, 0); err != nil {
		return Asset{}, err
	}
	for _, name := range files {
		path, err := findFile(unpacked, name)
		if err != nil {
			return Asset{}, err
		}
		if asset.Files[name], err = fileSHA256(path); err != nil {
			return Asset{}, err
		}
	}
	return asset, nil
}
