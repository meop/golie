// Package sheesh pins the release of sheesh — the shim templates and their
// kebab stamper — that this build of the library stamps shims with, and keeps
// a tool's vendored copy of it there.
package sheesh

import (
	"context"
	"runtime"

	"github.com/meop/golie/pinned"
)

// Version is the pinned sheesh release. A tool's vendored copy is synced to
// exactly this before it is used, and bumping it here, with the digests
// below, is the only way it moves: it is the tool's internals, not something
// its user chose to install.
const Version = "0.2.0"

// ReleaseBase is where sheesh's releases are downloaded from: a plain release
// download addressed by the pinned version, not the GitHub API. It is a var,
// not a const, so a test can point it at a local server.
var ReleaseBase = "https://github.com/meop/sheesh/releases/download/"

// Assets are the pinned release's archives, keyed by the "<os>-<arch>" sheesh
// spells in their names: each one's sha256, and that of every file kept from
// it. `go run ./cmd/pin` prints an entry; see AGENTS.md for bumping the pin.
var Assets = map[string]pinned.Asset{
	"linux-x86_64":    {SHA256: "b701a4e8d805c72698b9276105212f550968f10b13aa96ffd4a9e95b2c3896c3", Files: map[string]string{"kebab": "d44a7b9a93af4af68603149cb6a451f52b9c8955247525fc518f45fd270f91da", "sheesh": "95151e7d90b71144c3e33c8c53c8e3f1ac3453527c86f9ddbb6d4046037dd0da"}},
	"linux-aarch64":   {SHA256: "1c5e7b8379ba00d8b21816d1a1be390bb5845dbe9b5decc8a7dfd4b1b8d3dff0", Files: map[string]string{"kebab": "7b7f4cc0e239aef0dd1c755852c2137440a6c06d5b68bc0a32a06bada437fa2e", "sheesh": "3efa9d1fbc0f261c3f63c920ae588257ca3bbaa9f845eefb439b069f6c0e7779"}},
	"darwin-x86_64":   {SHA256: "642284b3f565be9eebc11e86022e346638d6a8c91263a1c2009fa3c0e0086f96", Files: map[string]string{"kebab": "11214a6f8b105295e54274851f1514d585784483db1cffbc685130c299b14ae5", "sheesh": "7a7f341b2fb3309981f28f7a913374428fb32b3503e122d502d9b280f82c06ab"}},
	"darwin-aarch64":  {SHA256: "32a3fc2a7f882568837e9c99f85657c497b720b9ab0ca5b53c0edecd4c47fe9c", Files: map[string]string{"kebab": "de38791c142513b5a3401b22b3481e46d32a03a687243d3e3f9ad9c11e6a8d58", "sheesh": "fcf45d0d374323e15cc95ef01cea3bdc328403a81571e5b21234e1d35a05f97d"}},
	"windows-x86_64":  {SHA256: "520d947326dcd247648caf5a0f22148cfc46d90f25645087249b2c6e24243af8", Files: map[string]string{"kebab.exe": "3550377085f0c4b938a1d2d8ea896ad6f07638a7cd2c84f9b070e07309a99925", "sheesh.exe": "65348b3e893ba430c63d9ee7e93b7a9ab8701d389f07304dd7cb9038a2786807", "sheesh-gui.exe": "00c750956c357fe553f1458efb2872ab3fee304a71f1b04a66cce8bac386ba5b"}},
	"windows-aarch64": {SHA256: "41d0b42b8c57df286846a15b9e8e64edf49c61c0b4e8309afa8c79f73dad2f0f", Files: map[string]string{"kebab.exe": "fd6e8ecc7c6b22faab5d563296da5b29efa3d1bda235609f737278004bd29a90", "sheesh.exe": "9ce8f4f51c9dbed53b04d9c8f72323193cbd1cfed39647e65114e55b5615e90b", "sheesh-gui.exe": "41ee77b334ed4b1843df7fce639a9be4141440d0f056834ac479d91f256d31c0"}},
}

// Tool is the pinned release as it applies to goos/goarch; ok is false where
// sheesh ships none.
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
	asset.URL = ReleaseBase + "v" + Version + "/sheesh-" + Version + "-" + key + ext
	return pinned.Tool{Name: "sheesh", Version: Version, Asset: asset}, true
}

// Sync makes dir hold the pinned sheesh for this machine. See pinned.Ensure.
func Sync(ctx context.Context, dir string, opts pinned.Options) error {
	tool, ok := Tool(runtime.GOOS, runtime.GOARCH)
	if !ok {
		return errNoRelease
	}
	return pinned.Ensure(ctx, tool, dir, opts)
}
