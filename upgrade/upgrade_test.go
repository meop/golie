package upgrade

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// selfArchive builds a release archive holding one file, gotm, as the release
// pipeline packs it for this platform.
func selfArchive(t *testing.T, body string) []byte {
	t.Helper()
	name := exeName("tool")
	var buf bytes.Buffer
	if runtime.GOOS == "windows" {
		zw := zip.NewWriter(&buf)
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	_, _ = tw.Write([]byte(body))
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// serveReleases stands up the tool's release page: "latest" redirects to tag, and
// the tag's download path serves the archive for this platform and a
// SHA256SUMS naming sum for it.
func serveReleases(t *testing.T, tag string, archive []byte, sum string) Self {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/latest", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/tag/"+tag, http.StatusFound)
	})
	mux.HandleFunc("/download/"+tag+"/SHA256SUMS", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0000  some-other-asset.tar.gz\n" + sum + "  " + (Self{Name: "tool"}).Asset(tag) + "\n"))
	})
	mux.HandleFunc("/download/"+tag+"/"+(Self{Name: "tool"}).Asset(tag), func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return Self{Name: "tool", ReleasesURL: srv.URL + "/"}
}

func fakeSelf(t *testing.T) string {
	t.Helper()
	self := filepath.Join(t.TempDir(), exeName("tool"))
	if err := os.WriteFile(self, []byte("old gotm"), 0755); err != nil {
		t.Fatal(err)
	}
	return self
}

func TestLatestTagReadsTheRedirect(t *testing.T) {
	s := serveReleases(t, "v1.2.3", nil, "")
	if tag, err := s.LatestTag(context.Background()); err != nil || tag != "v1.2.3" {
		t.Fatalf("latestTag = %q, %v", tag, err)
	}
}

func TestInstallSelfReplacesTheRunningBinary(t *testing.T) {
	self := fakeSelf(t)
	archive := selfArchive(t, "new gotm")
	sum := sha256.Sum256(archive)
	s := serveReleases(t, "v1.2.3", archive, hex.EncodeToString(sum[:]))
	s.Path = func() (string, error) { return self, nil }

	var fetched []string
	if err := s.Install(context.Background(), "v1.2.3", func(a string) { fetched = append(fetched, a) }); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(self); string(got) != "new gotm" {
		t.Errorf("the running binary is %q", got)
	}
	if len(fetched) != 1 || fetched[0] != s.Asset("v1.2.3") {
		t.Errorf("fetching was told %v", fetched)
	}
	entries, _ := os.ReadDir(filepath.Dir(self))
	for _, e := range entries {
		if e.Name() != filepath.Base(self) && !strings.HasSuffix(e.Name(), ".old") {
			t.Errorf("the upgrade left %s beside the binary", e.Name())
		}
	}
}

func TestInstallSelfRefusesAnArchiveThatDoesNotMatchItsChecksum(t *testing.T) {
	self := fakeSelf(t)
	s := serveReleases(t, "v1.2.3", selfArchive(t, "new gotm"), strings.Repeat("0", 64))
	s.Path = func() (string, error) { return self, nil }

	if err := s.Install(context.Background(), "v1.2.3", nil); err == nil {
		t.Fatal("an archive that does not match the release's checksum was installed")
	}
	if got, _ := os.ReadFile(self); string(got) != "old gotm" {
		t.Errorf("the running binary was replaced by an archive that failed its check: %q", got)
	}
}
