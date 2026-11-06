// Package wedge is the release of wedge — the shim templates and their mallet
// stamper — that a tool built on golie stamps its shims with: wedge's latest,
// vendored in the tool's own directory.
package wedge

import (
	"runtime"

	"github.com/meop/golie/sig"
	"github.com/meop/golie/vendored"
)

// ReleasesURL is wedge's release page. It is a var, not a const, so a test can
// point it at a local server.
var ReleasesURL = "https://github.com/meop/wedge/releases/"

// Key is the key wedge's SHA256SUMS is signed with. A var so a tool's own test
// can serve an unsigned release, with nil: the signature check itself is
// tested here, in vendored.
var Key = sig.ReleaseKey

// Release is wedge as it applies to goos/goarch: its archive there, the files
// kept from it, and the key its SHA256SUMS is signed with. ok is false where
// wedge ships none.
func Release(goos, goarch string) (vendored.Release, bool) {
	arch, ok := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[goarch]
	if !ok || !ships(goos, arch) {
		return vendored.Release{}, false
	}
	ext := ".tar.gz"
	files := []string{"mallet", "wedge"}
	if goos == "windows" {
		ext = ".zip"
		files = []string{"mallet.exe", "wedge.exe", "wedge-gui.exe"}
	}
	return vendored.Release{
		Name:        "wedge",
		ReleasesURL: ReleasesURL,
		Archive: func(version string) (string, bool) {
			return "wedge-" + version + "-" + goos + "-" + arch + ext, true
		},
		Sums:  func(string) string { return "SHA256SUMS" },
		Key:   Key,
		Files: files,
	}, true
}

// ships says whether wedge releases an archive for goos on arch: linux and
// windows on both, darwin on aarch64 only.
func ships(goos, arch string) bool {
	switch goos {
	case "linux", "windows":
		return true
	case "darwin":
		return arch == "aarch64"
	}
	return false
}

// Here is Release for this machine.
func Here() (vendored.Release, error) {
	r, ok := Release(runtime.GOOS, runtime.GOARCH)
	if !ok {
		return vendored.Release{}, errNoRelease
	}
	return r, nil
}
