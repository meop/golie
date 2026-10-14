package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestLayout(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".tool")
	l := Layout{Root: root}

	// naming a place an install would be does not make it
	for got, want := range map[string]string{
		l.ManifestPath():              filepath.Join(root, "manifest.json"),
		l.LockPath():                  filepath.Join(root, ".lock"),
		l.ExtractsDir():               filepath.Join(root, "extract"),
		l.ExtractBaseDir("fzf@14"):    filepath.Join(root, "extract", "fzf@14"),
		l.ExtractPath("fzf@14", "14"): filepath.Join(root, "extract", "fzf@14", "14"),
		l.DownloadsDir():              filepath.Join(root, "download"),
	} {
		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
	if exists(root) {
		t.Fatal("naming paths created the state directory")
	}

	bin, err := l.BinDir()
	if err != nil || bin != filepath.Join(root, "bin") || !exists(bin) {
		t.Errorf("BinDir = %s, %v", bin, err)
	}
	shims, err := l.ShimDir()
	if err != nil || shims != filepath.Join(root, "vendor", "sheesh") || !exists(shims) {
		t.Errorf("ShimDir = %s, %v", shims, err)
	}
	vendor, err := l.VendorDir()
	if err != nil || vendor != filepath.Join(root, "vendor") {
		t.Errorf("VendorDir = %s, %v", vendor, err)
	}
	extract, err := l.ExtractDir("go", "1.27.2")
	if err != nil || extract != l.ExtractPath("go", "1.27.2") || !exists(extract) {
		t.Errorf("ExtractDir = %s, %v", extract, err)
	}
	// a download dir is named by the version alone, whatever the tag carried
	download, err := l.DownloadDir("github.com/junegunn/fzf", "v0.70.0")
	if err != nil || download != filepath.Join(root, "download", "github.com", "junegunn", "fzf", "0.70.0") || !exists(download) {
		t.Errorf("DownloadDir = %s, %v", download, err)
	}
}
