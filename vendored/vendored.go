// Package vendored keeps a tool another tool vendors for its own use — wedge's
// mallet and shim templates, ghpm's gh — at that tool's latest release.
//
// Nothing here goes online behind the user's back. Ensure installs a tool only
// where there is none, as on a new machine; Refresh is the one call that asks
// which release is newest, for a command the user runs to mean exactly that.
// Either way a release is checked before any of it is put in place: its
// archive against the release's own checksum manifest, and that manifest
// against its signature when the tool names the key it is signed with.
package vendored

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/meop/golie/fetch"
	"github.com/meop/golie/ui"
	"github.com/meop/golie/upgrade"
)

// Release is where a vendored tool's releases are and how they are packed.
type Release struct {
	// Name is the tool's.
	Name string
	// ReleasesURL is its release page with a trailing slash:
	// "https://github.com/<owner>/<repo>/releases/". Tags are "v<version>".
	ReleasesURL string
	// Archive names the release's archive for this machine; ok is false
	// where the tool ships none.
	Archive func(version string) (name string, ok bool)
	// Sums names the release's checksum manifest.
	Sums func(version string) string
	// Key is the armored OpenPGP key the manifest is signed with, in
	// Sums(version)+".sig" beside it. Leave it nil for a release that is not
	// signed.
	Key []byte
	// Files are what is kept from the archive, each found by name wherever
	// it is nested.
	Files []string
}

// Options tunes Ensure and Refresh. All of it is optional.
type Options struct {
	// Keep names entries of the directory that are not the tool's files and
	// are to be left alone — its own configuration, say. Anything else found
	// there is removed, so nothing an older release shipped lingers.
	Keep []string
	// Fetching is called with the archive's name just before a download.
	Fetching func(archive string)
}

// VersionFile is the file that records which release a directory holds. It
// is written last, so a directory with it holds every file of that release.
const VersionFile = ".version"

// Installed is the release dir holds, or "" when it holds none whole.
func (r Release) Installed(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, VersionFile))
	if err != nil {
		return ""
	}
	for _, name := range r.Files {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return ""
		}
	}
	return strings.TrimSpace(string(data))
}

// Ensure makes dir hold the tool, asking nothing of the network when it
// already does: whatever release is there is used. Where none is, the latest
// is installed, and its version returned; "" means nothing was installed.
func (r Release) Ensure(ctx context.Context, dir string, opts Options) (string, error) {
	if have := r.Installed(dir); have != "" {
		ui.Verbose("%s %s: in %s", r.Name, have, dir)
		return "", nil
	}
	latest, err := r.Latest(ctx)
	if err != nil {
		return "", err
	}
	return latest, r.Install(ctx, dir, latest, opts)
}

// Refresh moves dir to the tool's latest release, and says what it had and
// what it has: the same twice when it was already current, and from "" when
// it held none.
func (r Release) Refresh(ctx context.Context, dir string, opts Options) (from, to string, err error) {
	from = r.Installed(dir)
	latest, err := r.Latest(ctx)
	if err != nil {
		return from, from, err
	}
	if latest == from {
		return from, from, nil
	}
	if err := r.Install(ctx, dir, latest, opts); err != nil {
		return from, from, err
	}
	return from, latest, nil
}

// Label is how a vendored tool is named in what a tool prints of it: its
// refresh, its downloads and its doctor all say `vendor - gh`.
func Label(name string) string { return "vendor - " + name }

// Said is a refresh's outcome as every tool that vendors one prints it, in
// the shape of the tools' other lines, label first and the detail in
// brackets: `vendor - gh: up to date [2.103.3]`,
// `vendor - gh: installed [2.103.3]`, `vendor - gh: updated [2.100.0 → 2.103.3]`.
// label is Label's for a vendored tool, or the caller's own for anything else
// a refresh moves.
func Said(label, from, to string) string {
	switch from {
	case to:
		return fmt.Sprintf("%s: up to date [%s]", label, to)
	case "":
		return fmt.Sprintf("%s: installed [%s]", label, to)
	default:
		return fmt.Sprintf("%s: updated [%s → %s]", label, from, to)
	}
}

// Latest is the version of the tool's newest release.
func (r Release) Latest(ctx context.Context) (string, error) {
	tag, err := fetch.LatestTag(ctx, r.ReleasesURL)
	if err != nil {
		return "", fmt.Errorf("%s: %w", r.Name, err)
	}
	return strings.TrimPrefix(tag, "v"), nil
}

// Install makes dir hold release version: every file of it, and nothing else
// beside what opts.Keep names. The archive is fetched and checked, and each
// file found in it, before any is put in place, so a release that fails its
// check fails the command and leaves what was there.
func (r Release) Install(ctx context.Context, dir, version string, opts Options) error {
	archiveName, ok := r.Archive(version)
	if !ok {
		return fmt.Errorf("%s %s: no release for this platform", r.Name, version)
	}
	base := r.ReleasesURL + "download/v" + version + "/"
	sum, err := upgrade.ReleaseSum(ctx, base, r.Sums(version), r.Key, archiveName)
	if err != nil {
		return fmt.Errorf("%s %s: %w", r.Name, version, err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// beside dir, not in it, so nothing of the fetch is ever among the files
	work, err := os.MkdirTemp(filepath.Dir(dir), "."+r.Name+"-sync-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(work) }()

	archive := filepath.Join(work, archiveName)
	ui.Verbose("%s %s: fetching %s", r.Name, version, base+archiveName)
	if opts.Fetching != nil {
		opts.Fetching(archiveName)
	}
	if err := fetch.Download(ctx, base+archiveName, archive, sum); err != nil {
		return fmt.Errorf("fetching %s %s: %w", r.Name, version, err)
	}
	unpacked := filepath.Join(work, "unpacked")
	if err := fetch.Extract(archive, unpacked, 0); err != nil {
		return err
	}
	found := make(map[string]string, len(r.Files))
	for _, name := range r.Files {
		path, err := findFile(unpacked, name)
		if err != nil {
			return fmt.Errorf("%s %s: %w", r.Name, version, err)
		}
		found[name] = path
	}

	// gone first, so a directory caught halfway is one Installed rejects
	if err := os.Remove(filepath.Join(dir, VersionFile)); err != nil && !os.IsNotExist(err) {
		return err
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
	if err := r.sweep(dir, opts.Keep); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, VersionFile), []byte(version+"\n"), 0644); err != nil {
		return err
	}
	ui.Verbose("%s %s: installed in %s", r.Name, version, dir)
	return nil
}

// sweep removes what dir holds that is neither one of the tool's files nor
// kept.
func (r Release) sweep(dir string, keep []string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if slices.Contains(r.Files, name) || slices.Contains(keep, name) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, name)); err != nil {
			return err
		}
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
		return "", fmt.Errorf("%s not found in the archive", name)
	}
	return found, nil
}
