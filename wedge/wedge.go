// Package wedge pins the release of wedge — the shim templates and their
// mallet stamper — that this build of the library stamps shims with, and keeps
// a tool's vendored copy of it there.
package wedge

import (
	"context"
	"runtime"

	"github.com/meop/golie/pinned"
)

// Version is the pinned wedge release. A tool's vendored copy is synced to
// exactly this before it is used, and bumping it here, with the digests
// below, is the only way it moves: it is the tool's internals, not something
// its user chose to install.
const Version = "0.5.2"

// ReleaseBase is where wedge's releases are downloaded from: a plain release
// download addressed by the pinned version, not the GitHub API. It is a var,
// not a const, so a test can point it at a local server.
var ReleaseBase = "https://github.com/meop/wedge/releases/download/"

// Assets are the pinned release's archives, keyed by the "<os>-<arch>" wedge
// spells in their names: each one's sha256, and that of every file kept from
// it. `go run ./cmd/pin` prints an entry; see AGENTS.md for bumping the pin.
var Assets = map[string]pinned.Asset{
	"linux-x86_64":    {SHA256: "207ac684d0e2e7e5a0046a02b83a37bc26e4cdbd556fbbdbc605137eabc582b9", Files: map[string]string{"mallet": "e05e53bd38d2bbecc3f6079e34b83a3c47ed6943af44a902a9e91ec0d9f33221", "wedge": "77f4de28102ee460813aa2837ed38000409a379a4ebdfa56f6e92206a390e962"}},
	"linux-aarch64":   {SHA256: "971245e100da9ff59edbe32689940894263ecaa5a82166cca2a32275475ddc76", Files: map[string]string{"mallet": "595b19d82e07bc15b8d31dcc99e7de4d6d76960c11a021c38c7402fbcee20314", "wedge": "2afd02e8c8fff8a4f9a8dfebaedec9b58c6997455bdb69cff5b61d4febcd09d1"}},
	"darwin-x86_64":   {SHA256: "b1f294df171466ad53c5fae07fc412c4a415dc7d0b1349d115d87a4a642acb6d", Files: map[string]string{"mallet": "15dbc92d03b71a699808889fa3ee0032d4c767a9feba56956bfce00b0a564f23", "wedge": "dbdee0d509a21805b9e4d96b4c00a3bd88865238c11d470df13fd5c93a3d0437"}},
	"darwin-aarch64":  {SHA256: "da91b564e541a7244fc13b52f2942a73633b8d9eff4c35d387de7847f74b33d6", Files: map[string]string{"mallet": "06c25efb2e0ba22ce4e8d71c3127d95f6bff72370abed439b56daacd51df6338", "wedge": "0e17fda22c82db7a00284a0b0b2e8354eec6358760dcdc37507287b19a6954ab"}},
	"windows-x86_64":  {SHA256: "92f613045d8bcaa8c9202f68ebb70cb3e5aeaf600b631abf93cd9874d993b6ee", Files: map[string]string{"mallet.exe": "63255779188c01506ef70905a8cd62be0efaaf2fdc8f7854a5921d47e3f9d722", "wedge.exe": "0aa7e65cec784e236bd5ed928f0a288b0cc9ee093c3fddb2385eb3a5eeb1d38f", "wedge-gui.exe": "1c7461947360f4240a290ef9d14ce22ad58ebd9cc78308fc5c8623486aa51287"}},
	"windows-aarch64": {SHA256: "aea6252c28cfe12f4c5fdde4c63ac12691e783fb5c8aad3cfcdfa5573bf05409", Files: map[string]string{"mallet.exe": "e568deee397c3c0f89c6e1a890a14a47364aacae148c4fa39a0eb4c534ec4350", "wedge.exe": "8a38cadc3ec88e5929d5f203d47b0e48f944d3f3aad6a32815c98a589f67a58c", "wedge-gui.exe": "9d13cbb26df30fa9e57e58cb6fd7037a6e81f47690a815ba61ae7c794d2b9dcc"}},
}

// Tool is the pinned release as it applies to goos/goarch; ok is false where
// wedge ships none.
func Tool(goos, goarch string) (pinned.Tool, bool) {
	arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[goarch]
	key := goos + "-" + arch
	asset, ok := Assets[key]
	if !ok {
		return pinned.Tool{}, false
	}
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	asset.URL = ReleaseBase + "v" + Version + "/wedge-" + Version + "-" + key + ext
	return pinned.Tool{Name: "wedge", Version: Version, Asset: asset}, true
}

// Sync makes dir hold the pinned wedge for this machine. See pinned.Ensure.
func Sync(ctx context.Context, dir string, opts pinned.Options) error {
	tool, ok := Tool(runtime.GOOS, runtime.GOARCH)
	if !ok {
		return errNoRelease
	}
	return pinned.Ensure(ctx, tool, dir, opts)
}
