package pinned

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func sum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// archive builds a tar.gz nesting each file under a versioned folder, the way
// a release packs them.
func archive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: "tool-1.0/" + name, Mode: 0755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// serve offers data as a release archive and returns the tool pinned to it
// and a counter of how often it was fetched.
func serve(t *testing.T, shipped map[string]string, pinnedFiles map[string]string) (Tool, *int) {
	t.Helper()
	data := archive(t, shipped)
	fetches := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches++
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	files := map[string]string{}
	for name, body := range pinnedFiles {
		files[name] = sum([]byte(body))
	}
	return Tool{Name: "tool", Version: "1.0", Asset: Asset{URL: srv.URL + "/tool-1.0.tar.gz", SHA256: sum(data), Files: files}}, &fetches
}

func entries(t *testing.T, dir string) []string {
	t.Helper()
	list, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range list {
		names = append(names, e.Name())
	}
	slices.Sort(names)
	return names
}

var shipped = map[string]string{"tool": "the tool", "helper": "its helper", "README": "not kept"}
var kept = map[string]string{"tool": "the tool", "helper": "its helper"}

func TestEnsureVendorsThePinnedFilesAndNothingElse(t *testing.T) {
	tool, fetches := serve(t, shipped, kept)
	vendor := t.TempDir()
	dir := filepath.Join(vendor, "tool")

	if err := Ensure(context.Background(), tool, dir, Options{}); err != nil {
		t.Fatal(err)
	}
	if got := entries(t, dir); !slices.Equal(got, []string{".pin", "helper", "tool"}) {
		t.Errorf("dir holds %v", got)
	}
	if got := entries(t, vendor); !slices.Equal(got, []string{"tool"}) {
		t.Errorf("the sync left %v beside the tool's dir", got)
	}
	if !tool.Verified(dir) {
		t.Error("a fresh sync does not verify")
	}

	// in step with its pin: a hash per file, and no fetch
	if err := Ensure(context.Background(), tool, dir, Options{}); err != nil || *fetches != 1 {
		t.Errorf("a second Ensure returned %v after %d fetches, want none more", err, *fetches)
	}
}

// a copy from another version, a damaged one and a swapped one all fail the
// same check; only one that claims this very version is worth saying
func TestEnsureReplacesWhatDoesNotMatch(t *testing.T) {
	for name, stamp := range map[string]string{"another version": "0.9", "this version": "1.0"} {
		tool, _ := serve(t, shipped, kept)
		dir := filepath.Join(t.TempDir(), "tool")
		if err := os.MkdirAll(filepath.Join(dir, "config"), 0755); err != nil {
			t.Fatal(err)
		}
		for file, body := range map[string]string{"tool": "not the tool", "helper": "its helper", "shipped-by-an-older-release": "x", ".pin": stamp + "\n", "config/token": "secret"} {
			if err := os.WriteFile(filepath.Join(dir, file), []byte(body), 0755); err != nil {
				t.Fatal(err)
			}
		}
		var fetched, replacing []string
		opts := Options{
			Keep:      []string{"config"},
			Fetching:  func(a string) { fetched = append(fetched, a) },
			Replacing: func(f string) { replacing = append(replacing, f) },
		}
		if err := Ensure(context.Background(), tool, dir, opts); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got, _ := os.ReadFile(filepath.Join(dir, "tool")); string(got) != "the tool" {
			t.Errorf("%s: tool is %q", name, got)
		}
		if got := entries(t, dir); !slices.Equal(got, []string{".pin", "config", "helper", "tool"}) {
			t.Errorf("%s: dir holds %v", name, got)
		}
		if got, _ := os.ReadFile(filepath.Join(dir, "config", "token")); string(got) != "secret" {
			t.Errorf("%s: a kept entry was disturbed", name)
		}
		if !slices.Equal(fetched, []string{"tool-1.0.tar.gz"}) {
			t.Errorf("%s: fetching was told %v", name, fetched)
		}
		if wantSaid := stamp == "1.0"; (len(replacing) == 1) != wantSaid {
			t.Errorf("%s: replacing was told %v", name, replacing)
		}
	}
}

func TestEnsureRefusesAReleaseThatDoesNotMatchItsPin(t *testing.T) {
	prepare := func(t *testing.T) string {
		dir := filepath.Join(t.TempDir(), "tool")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "tool"), []byte("the old tool"), 0755); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	check := func(t *testing.T, tool Tool, dir string) {
		if err := Ensure(context.Background(), tool, dir, Options{}); err == nil {
			t.Fatal("a release that does not match its pin was vendored")
		}
		if got, _ := os.ReadFile(filepath.Join(dir, "tool")); string(got) != "the old tool" {
			t.Errorf("what was vendored before was disturbed: %q", got)
		}
	}
	t.Run("the archive", func(t *testing.T) {
		tool, _ := serve(t, shipped, kept)
		tool.Asset.SHA256 = strings.Repeat("0", 64)
		check(t, tool, prepare(t))
	})
	t.Run("a file in it", func(t *testing.T) {
		tool, _ := serve(t, shipped, map[string]string{"tool": "the tool", "helper": "a different helper"})
		check(t, tool, prepare(t))
	})
	t.Run("a file it lacks", func(t *testing.T) {
		tool, _ := serve(t, shipped, map[string]string{"tool": "the tool", "absent": "x"})
		check(t, tool, prepare(t))
	})
}

func TestReadPinsAnArchive(t *testing.T) {
	tool, _ := serve(t, shipped, kept)
	asset, err := Read(context.Background(), tool.Asset.URL, []string{"tool", "helper"})
	if err != nil {
		t.Fatal(err)
	}
	if asset.SHA256 != tool.Asset.SHA256 || asset.Files["tool"] != tool.Asset.Files["tool"] || asset.Files["helper"] != tool.Asset.Files["helper"] {
		t.Errorf("Read = %+v, want %+v", asset, tool.Asset)
	}
}
