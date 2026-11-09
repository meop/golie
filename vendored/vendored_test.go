package vendored

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
	"testing"

	"github.com/meop/golie/internal/testsign"
)

// archive builds a tar.gz nesting each file under a versioned folder, the way
// a release packs them.
func archive(t *testing.T, version string, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: "tool-" + version + "/" + name, Mode: 0755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
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

// releases is a release page the test can move to a new latest
type releases struct {
	latest   string
	archives map[string][]byte
	// sums is what each release's manifest lists for its archive, when not
	// the archive's own sha256
	sums    map[string]string
	sign    func([]byte) []byte
	fetches int
}

func (rs *releases) serve(t *testing.T, key []byte) Release {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/latest", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/tag/v"+rs.latest, http.StatusFound)
	})
	manifest := func(version string) []byte {
		sum := rs.sums[version]
		if sum == "" {
			s := sha256.Sum256(rs.archives[version])
			sum = hex.EncodeToString(s[:])
		}
		return []byte(sum + "  tool-" + version + ".tar.gz\n")
	}
	mux.HandleFunc("/download/{tag}/{file}", func(w http.ResponseWriter, r *http.Request) {
		version := r.PathValue("tag")[1:]
		switch r.PathValue("file") {
		case "SHA256SUMS":
			_, _ = w.Write(manifest(version))
		case "SHA256SUMS.sig":
			if rs.sign == nil {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(rs.sign(manifest(version)))
		case "tool-" + version + ".tar.gz":
			rs.fetches++
			_, _ = w.Write(rs.archives[version])
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return Release{
		Name:        "tool",
		ReleasesURL: srv.URL + "/",
		Archive:     func(v string) (string, bool) { return "tool-" + v + ".tar.gz", true },
		Sums:        func(string) string { return "SHA256SUMS" },
		Key:         key,
		Files:       []string{"tool", "helper"},
	}
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

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func twoReleases(t *testing.T) *releases {
	return &releases{latest: "1.0", archives: map[string][]byte{
		"1.0": archive(t, "1.0", map[string]string{"tool": "tool 1.0", "helper": "helper 1.0", "README": "not kept"}),
		"1.1": archive(t, "1.1", map[string]string{"tool": "tool 1.1", "helper": "helper 1.1"}),
	}}
}

func TestEnsureInstallsTheLatestWhereThereIsNoneAndThenStaysOffline(t *testing.T) {
	rs := twoReleases(t)
	r := rs.serve(t, nil)
	vendor := t.TempDir()
	dir := filepath.Join(vendor, "tool")

	got, err := r.Ensure(context.Background(), dir, Options{})
	if err != nil || got != "1.0" {
		t.Fatalf("Ensure = %q, %v", got, err)
	}
	if got := entries(t, dir); !slices.Equal(got, []string{".version", "helper", "tool"}) {
		t.Errorf("dir holds %v", got)
	}
	if got := entries(t, vendor); !slices.Equal(got, []string{"tool"}) {
		t.Errorf("the install left %v beside the tool's dir", got)
	}

	// a newer release is not looked for: Ensure keeps what is there
	rs.latest = "1.1"
	got, err = r.Ensure(context.Background(), dir, Options{})
	if err != nil || got != "" || rs.fetches != 1 || read(t, filepath.Join(dir, "tool")) != "tool 1.0" {
		t.Errorf("a second Ensure = %q, %v after %d fetches, want nothing done", got, err, rs.fetches)
	}
}

func TestRefreshMovesToTheLatestAndKeepsWhatItIsTold(t *testing.T) {
	rs := twoReleases(t)
	r := rs.serve(t, nil)
	dir := filepath.Join(t.TempDir(), "tool")
	if _, err := r.Ensure(context.Background(), dir, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"config/token": "secret", ".pin": "0.9\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}

	from, to, err := r.Refresh(context.Background(), dir, Options{Keep: []string{"config"}})
	if err != nil || from != "1.0" || to != "1.0" || rs.fetches != 1 {
		t.Fatalf("Refresh while current = %q → %q, %v after %d fetches", from, to, err, rs.fetches)
	}

	rs.latest = "1.1"
	var fetched []string
	from, to, err = r.Refresh(context.Background(), dir, Options{Keep: []string{"config"}, Fetching: func(a string) { fetched = append(fetched, a) }})
	if err != nil || from != "1.0" || to != "1.1" {
		t.Fatalf("Refresh = %q → %q, %v", from, to, err)
	}
	if read(t, filepath.Join(dir, "tool")) != "tool 1.1" || read(t, filepath.Join(dir, "helper")) != "helper 1.1" {
		t.Error("the files are not the new release's")
	}
	if got := entries(t, dir); !slices.Equal(got, []string{".version", "config", "helper", "tool"}) {
		t.Errorf("dir holds %v", got)
	}
	if read(t, filepath.Join(dir, "config", "token")) != "secret" {
		t.Error("a kept entry was disturbed")
	}
	if !slices.Equal(fetched, []string{"tool-1.1.tar.gz"}) {
		t.Errorf("fetching was told %v", fetched)
	}
}

func TestADirectoryMissingAFileHoldsNoRelease(t *testing.T) {
	rs := twoReleases(t)
	r := rs.serve(t, nil)
	dir := filepath.Join(t.TempDir(), "tool")
	if _, err := r.Ensure(context.Background(), dir, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "helper")); err != nil {
		t.Fatal(err)
	}
	if got := r.Installed(dir); got != "" {
		t.Fatalf("Installed = %q with a file gone", got)
	}
	if got, err := r.Ensure(context.Background(), dir, Options{}); err != nil || got != "1.0" {
		t.Errorf("Ensure = %q, %v, want the release put back", got, err)
	}
}

func TestARefreshThatFailsItsCheckLeavesWhatWasThere(t *testing.T) {
	key, sign := testsign.New(t)
	_, otherSign := testsign.New(t)
	for name, spoil := range map[string]func(rs *releases){
		"an archive its manifest does not list": func(rs *releases) { rs.sums["1.1"] = "0000" },
		"a manifest signed by another key":      func(rs *releases) { rs.sign = otherSign },
		"a manifest with no signature":          func(rs *releases) { rs.sign = nil },
		"an archive without a file":             func(rs *releases) { rs.archives["1.1"] = archive(t, "1.1", map[string]string{"tool": "tool 1.1"}) },
	} {
		t.Run(name, func(t *testing.T) {
			rs := twoReleases(t)
			rs.sums = map[string]string{}
			rs.sign = sign
			r := rs.serve(t, key)
			dir := filepath.Join(t.TempDir(), "tool")
			if _, err := r.Ensure(context.Background(), dir, Options{}); err != nil {
				t.Fatal(err)
			}
			rs.latest = "1.1"
			spoil(rs)

			from, to, err := r.Refresh(context.Background(), dir, Options{})
			if err == nil || from != "1.0" || to != "1.0" {
				t.Fatalf("Refresh = %q → %q, %v", from, to, err)
			}
			if r.Installed(dir) != "1.0" || read(t, filepath.Join(dir, "tool")) != "tool 1.0" {
				t.Error("what was there was disturbed")
			}
		})
	}
}

func TestSaidNamesTheToolFirstAndTheVersionsInBrackets(t *testing.T) {
	for _, c := range []struct{ from, to, want string }{
		{"2.103.3", "2.103.3", "vendor - gh: up to date [2.103.3]"},
		{"", "2.103.3", "vendor - gh: installed [2.103.3]"},
		{"2.100.0", "2.103.3", "vendor - gh: updated [2.100.0 → 2.103.3]"},
	} {
		if got := Said(Label("gh"), c.from, c.to); got != c.want {
			t.Errorf("Said(Label(gh), %q, %q) = %q, want %q", c.from, c.to, got, c.want)
		}
	}
}
