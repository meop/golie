// Package layout names the directories of a tool's state directory
// (~/.ghpm, ~/.gotm). Both tools lay theirs out the same way, which is what
// lets one tidy clean either.
//
//	bin/                          the tool and its shims; the one dir on PATH
//	vendor/wedge/                mallet and its templates; never on PATH
//	extract/<key>/                one install, unpacked, at the same path from one version to the next
//	extract/.<key>.*              an install on its way in or out; never there between ops
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

// ExtractPath is where one install is unpacked, whether or not it is there.
func (l Layout) ExtractPath(key string) string { return filepath.Join(l.ExtractsDir(), key) }

// StagePath is where the next unpack of key is made, for journal.Swap.
func (l Layout) StagePath(key string) string {
	return filepath.Join(l.ExtractsDir(), "."+key+".new")
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
