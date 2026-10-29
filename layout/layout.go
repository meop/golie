// Package layout names the directories of a tool's state directory
// (~/.ghpm, ~/.gotm). Both tools lay theirs out the same way, which is what
// lets one tidy clean either.
//
//	bin/                          the tool and its shims; the one dir on PATH
//	vendor/wedge/                mallet and its templates; never on PATH
//	extract/<key>/<version>/      one version of one install, unpacked
//	extract/<key>/latest          a pointer file to the version installed, which shims are stamped through
//	extract/<key>/.<version>.new  a version being unpacked; never there between ops
//	download/<source>/<version>/  the release assets it was unpacked from
//	manifest.json, .lock
package layout

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/meop/golie/vers"
)

// Layout is a state directory rooted at Root.
type Layout struct{ Root string }

func (l Layout) made(elems ...string) (string, error) {
	dir := filepath.Join(append([]string{l.Root}, elems...)...)
	return dir, os.MkdirAll(dir, 0755)
}

// ManifestPath is the manifest file.
func (l Layout) ManifestPath() string { return filepath.Join(l.Root, "manifest.json") }

// LockPath is the file the tool's one-process-at-a-time lock is held on.
func (l Layout) LockPath() string { return filepath.Join(l.Root, ".lock") }

// BinDir holds the tool itself and the shims it stamps: everything meant to
// be on the user's PATH. It is created.
func (l Layout) BinDir() (string, error) { return l.made("bin") }

// VendorDir holds the tools the tool needs to work, not the ones it installs
// for you. Nothing here belongs on PATH. It is created.
func (l Layout) VendorDir() (string, error) { return l.made("vendor") }

// ShimDir holds wedge's mallet stamper and its templates. It is created.
func (l Layout) ShimDir() (string, error) { return l.made("vendor", "wedge") }

// ExtractsDir is where installs are unpacked. It is not created: it existing
// says something was installed.
func (l Layout) ExtractsDir() string { return filepath.Join(l.Root, "extract") }

// ExtractPath holds every version of one install, and its latest pointer.
func (l Layout) ExtractPath(key string) string { return filepath.Join(l.ExtractsDir(), key) }

// LatestPath is the pointer file to the version of key installed, which shims are stamped through.
func (l Layout) LatestPath(key string) string { return filepath.Join(l.ExtractPath(key), Latest) }

// Latest is the name of an install's pointer to its version.
const Latest = "latest"

// VersionPath is where one version of key is unpacked.
func (l Layout) VersionPath(key, version string) string {
	return filepath.Join(l.ExtractPath(key), version)
}

// StagePath is where a version of key is unpacked, for journal.Place.
func (l Layout) StagePath(key, version string) string {
	return filepath.Join(l.ExtractPath(key), "."+version+".new")
}

// DownloadsDir is the cache of release assets. It is not created.
func (l Layout) DownloadsDir() string { return filepath.Join(l.Root, "download") }

// DownloadDir holds the assets of one release of one source. It is created.
func (l Layout) DownloadDir(source, version string) (string, error) {
	dir := filepath.Join(l.DownloadsDir(), SourceToRelPath(source), vers.Normalize(version))
	return dir, os.MkdirAll(dir, 0755)
}

// SourceToRelPath turns a source ("github.com/owner/repo") into a relative
// filesystem path.
func SourceToRelPath(source string) string {
	return strings.ReplaceAll(source, "/", string(filepath.Separator))
}
