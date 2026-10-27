// Package pinned keeps a tool another tool vendors for its own use at exactly
// the release its build pins.
//
// A pin is a version and the sha256 of everything that version puts on disk:
// the archive it ships in, and each file kept from it. The files are checked
// by content every time they are about to be used, so what runs is what the
// build was made to run — a copy from another version, a damaged one, and one
// something else put there all fail the same check and are replaced the same
// way, from a URL the pin names and never from a "latest".
package pinned

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/meop/golie/fetch"
	"github.com/meop/golie/ui"
)

// Asset is the archive of a pinned release for one platform.
type Asset struct {
	// URL is where the archive is downloaded from: a plain download, not an
	// API call.
	URL string
	// SHA256 is the archive's.
	SHA256 string
	// Files maps the name of each file kept from the archive to its sha256.
	// A file is found in the archive by this name, wherever it is nested.
	Files map[string]string
}

// Tool is a pinned release as it applies to this machine.
type Tool struct {
	Name    string
	Version string
	Asset   Asset
}

// Options tunes Ensure. All of it is optional.
type Options struct {
	// Keep names entries of the directory that are not the tool's files and
	// are to be left alone — its own configuration, say. Anything else found
	// there is removed, so nothing an older release shipped lingers.
	Keep []string
	// Fetching is called with the archive's name just before a download.
	Fetching func(archive string)
	// Replacing is called, before a fetch, when the files were there for this
	// very version and did not match its pin: not a version to move from, but
	// a copy that was damaged or changed.
	Replacing func(file string)
}

// stampName is the file that records which version a directory was last
// synced to. It is not trusted for anything: the files' contents are the
// check. It only tells a copy of another version, which is the ordinary
// reason to replace, from this version's copy gone wrong, which is worth
// saying.
const stampName = ".pin"

// Verified reports whether dir holds every file of t with exactly its pinned
// contents.
func (t Tool) Verified(dir string) bool {
	_, ok := t.mismatch(dir)
	return ok
}

// mismatch returns the first file of t that dir does not hold with its pinned
// contents, and ok when there is none.
func (t Tool) mismatch(dir string) (file string, ok bool) {
	names := make([]string, 0, len(t.Asset.Files))
	for name := range t.Asset.Files {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		got, err := fileSHA256(filepath.Join(dir, name))
		if err != nil || !strings.EqualFold(got, t.Asset.Files[name]) {
			return name, false
		}
	}
	return "", len(names) > 0
}

// Ensure makes dir hold t: every pinned file with its pinned contents, and
// nothing else beside what opts.Keep names. When it already does, that is one
// hash per file and no network. When it does not, the archive is fetched from
// the pin's URL and checked, each file kept from it is checked against its
// own pin, and only then are they put in place — so a release that does not
// match its pin fails the command rather than being installed, and whatever
// was there before stays.
func Ensure(ctx context.Context, t Tool, dir string, opts Options) error {
	file, ok := t.mismatch(dir)
	if ok {
		ui.Verbose("%s %s: current in %s", t.Name, t.Version, dir)
		return nil
	}
	if len(t.Asset.Files) == 0 {
		return fmt.Errorf("%s %s pins no files for this platform", t.Name, t.Version)
	}
	had := stampedVersion(dir)
	if had == "" {
		had = "none"
	}
	ui.Verbose("%s %s: %s has %s, syncing", t.Name, t.Version, dir, had)
	if had == t.Version {
		if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			ui.Verbose("%s %s: %s does not match its pin", t.Name, t.Version, file)
			if opts.Replacing != nil {
				opts.Replacing(file)
			}
		}
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// beside dir, not in it, so nothing of the fetch is ever among the files
	work, err := os.MkdirTemp(filepath.Dir(dir), "."+t.Name+"-sync-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(work) }()

	archive := filepath.Join(work, filepath.Base(t.Asset.URL))
	ui.Verbose("%s %s: fetching %s", t.Name, t.Version, t.Asset.URL)
	if opts.Fetching != nil {
		opts.Fetching(filepath.Base(archive))
	}
	if err := fetch.Download(ctx, t.Asset.URL, archive, t.Asset.SHA256); err != nil {
		return fmt.Errorf("fetching %s %s: %w", t.Name, t.Version, err)
	}
	unpacked := filepath.Join(work, "unpacked")
	if err := fetch.Extract(archive, unpacked, 0); err != nil {
		return err
	}

	// every file is found and checked before any is placed
	found := make(map[string]string, len(t.Asset.Files))
	for name, want := range t.Asset.Files {
		path, err := findFile(unpacked, name)
		if err != nil {
			return fmt.Errorf("%s %s: %w", t.Name, t.Version, err)
		}
		got, err := fileSHA256(path)
		if err != nil {
			return err
		}
		if !strings.EqualFold(got, want) {
			return fmt.Errorf("%s %s: %s in the release does not match its pin", t.Name, t.Version, name)
		}
		found[name] = path
	}

	for name, path := range found {
		dest := filepath.Join(dir, name)
		// dropped first so the rename is not replacing a file another process
		// may still hold open
		_ = os.Remove(dest)
		if err := os.Rename(path, dest); err != nil {
			return err
		}
		if err := os.Chmod(dest, 0755); err != nil {
			return err
		}
	}
	if err := sweep(dir, t, opts.Keep); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, stampName), []byte(t.Version+"\n"), 0644); err != nil {
		return err
	}
	ui.Verbose("%s %s: installed in %s", t.Name, t.Version, dir)
	return nil
}

// sweep removes what dir holds that is neither one of t's files nor kept.
func sweep(dir string, t Tool, keep []string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if _, ok := t.Asset.Files[name]; ok || name == stampName || slices.Contains(keep, name) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	return nil
}

func stampedVersion(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, stampName))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// FileSHA256 is the sha256 of the file at path, in hex.
func FileSHA256(path string) (string, error) { return fileSHA256(path) }

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// findFile returns the path of the file called name somewhere under dir.
func findFile(dir, name string) (string, error) {
	var found string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == name {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("%s not found in the archive", name)
	}
	return found, nil
}
