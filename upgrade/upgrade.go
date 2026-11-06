// Package upgrade is a tool upgrading itself from its own GitHub releases.
//
// Everything it asks of GitHub is a plain page or release download — never
// the API, which is rate limited and would want a token, and never the gh
// CLI. The newest release is read from where the release page's "latest"
// address redirects, and an archive is checked against that release's own
// SHA256SUMS before the running binary is replaced.
//
// A tool that names the directory its installer puts it in upgrades only the
// copy there. One that runs from anywhere else was put there by something
// else — a package manager, go install, a build — and that is what upgrades
// it: replacing it in place would leave whatever installed it describing a
// binary that is no longer the one it installed.
package upgrade

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/meop/golie/fetch"
	"github.com/meop/golie/sig"
)

// Self is a tool that upgrades itself.
type Self struct {
	// Name is the tool's, and its binary's without an executable suffix.
	Name string
	// ReleasesURL is the tool's release page with a trailing slash:
	// "https://github.com/<owner>/<repo>/releases/".
	ReleasesURL string
	// Dir is the directory the tool's own installer puts it in, and the only
	// place it upgrades itself from. Leave it empty for a tool that upgrades
	// wherever it runs.
	Dir string
	// Path returns the running binary. Leave it nil for the process's own
	// executable; a test names a file of its own.
	Path func() (string, error)
	// Key is the armored OpenPGP key the release's SHA256SUMS is signed with,
	// in SHA256SUMS.sig beside it: sig.ReleaseKey for a meop tool. Leave it
	// nil for a release that is not signed.
	Key []byte
}

// NotSelfInstalledError is a tool refusing to upgrade a copy of itself that
// its own installer did not place.
type NotSelfInstalledError struct {
	// Running is the binary that was asked to upgrade, and Installed where
	// its installer would have put it.
	Running, Installed string
}

func (e *NotSelfInstalledError) Error() string {
	return fmt.Sprintf("not self installed: running from %s, not %s", e.Running, e.Installed)
}

// Check says whether the running binary is one Install may replace: any, for
// a tool that names no Dir, and otherwise only the one in Dir, however it was
// reached. It is a *NotSelfInstalledError when not, for the tool to say in
// its own words. It asks nothing of the network, so a tool calls it before it
// asks which release is newest.
func (s Self) Check() error {
	_, err := s.path()
	return err
}

func (s Self) running() (string, error) {
	if s.Path != nil {
		return s.Path()
	}
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(self)
}

// path is the running binary, once it is known to be the one in Dir.
func (s Self) path() (string, error) {
	self, err := s.running()
	if err != nil || s.Dir == "" {
		return self, err
	}
	installed := filepath.Join(s.Dir, exeName(s.Name))
	if !sameFile(self, installed) {
		return "", &NotSelfInstalledError{Running: self, Installed: installed}
	}
	return self, nil
}

// sameFile says whether two paths name one file. It is asked of the files
// and not their names, so a symlink to the installed binary, or a path spelt
// in another case on Windows, is still that binary; a path with nothing at it
// is no file at all.
func sameFile(a, b string) bool {
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	return err == nil && os.SameFile(ai, bi)
}

func exeName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// LatestTag asks which release is newest. See fetch.LatestTag.
func (s Self) LatestTag(ctx context.Context) (string, error) {
	tag, err := fetch.LatestTag(ctx, s.ReleasesURL)
	if err != nil {
		return "", fmt.Errorf("%s has no release to upgrade to: %w", s.Name, err)
	}
	return tag, nil
}

// Asset is the archive of release tag for this machine, as a goreleaser
// pipeline names it: "<name>-<version>-<os>-<arch>", a zip on windows and a
// tar.gz elsewhere.
func (s Self) Asset(tag string) string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return s.Name + "-" + strings.TrimPrefix(tag, "v") + "-" + runtime.GOOS + "-" + runtime.GOARCH + ext
}

// Install fetches release tag's archive for this machine, checks it against
// the release's own checksum manifest, and puts the binary in it where the
// running one is. Nothing is fetched for a copy Check refuses. fetching, when
// not nil, is called with the archive's name just before the download.
func (s Self) Install(ctx context.Context, tag string, fetching func(asset string)) error {
	self, err := s.path()
	if err != nil {
		return err
	}
	base := s.ReleasesURL + "download/" + tag + "/"
	asset := s.Asset(tag)
	sum, err := ReleaseSum(ctx, base, "SHA256SUMS", s.Key, asset)
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp(filepath.Dir(self), "."+s.Name+"-upgrade-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(work) }()

	archive := filepath.Join(work, asset)
	if fetching != nil {
		fetching(asset)
	}
	if err := fetch.Download(ctx, base+asset, archive, sum); err != nil {
		return err
	}
	unpacked := filepath.Join(work, "unpacked")
	if err := fetch.Extract(archive, unpacked, 0); err != nil {
		return err
	}
	fresh, err := findFile(unpacked, exeName(s.Name))
	if err != nil {
		return fmt.Errorf("%s holds no %s", asset, s.Name)
	}
	if err := os.Chmod(fresh, 0755); err != nil {
		return err
	}
	return replaceSelf(fresh, self)
}

// ReleaseSum reads the checksum manifest sums from a release's download
// path base, checks it against its detached signature sums+".sig" when key
// is not nil, and returns the sha256 it lists for asset.
func ReleaseSum(ctx context.Context, base, sums string, key []byte, asset string) (string, error) {
	manifest, err := fetch.Get(ctx, base+sums)
	if err != nil {
		return "", err
	}
	if key != nil {
		signature, err := fetch.Get(ctx, base+sums+".sig")
		if err != nil {
			return "", err
		}
		if err := sig.Verify(key, manifest, signature); err != nil {
			return "", fmt.Errorf("%s: %w", base+sums, err)
		}
	}
	return fetch.SumFor(manifest, asset)
}

// replaceSelf moves src to dst, the running binary. Windows will not let a
// running executable be overwritten, but it will let it be renamed: the old
// one is moved aside first, and put back if the new one will not go in. What
// is left aside is removed by the next upgrade, once nothing is running it.
func replaceSelf(src, dst string) error {
	if runtime.GOOS != "windows" {
		return os.Rename(src, dst)
	}
	old := dst + ".old"
	_ = os.Remove(old)
	if err := os.Rename(dst, old); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		return errors.Join(err, os.Rename(old, dst))
	}
	return nil
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
		return "", fmt.Errorf("%s not found", name)
	}
	return found, nil
}
