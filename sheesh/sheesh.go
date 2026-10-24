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
const Version = "0.2.1"

// ReleaseBase is where sheesh's releases are downloaded from: a plain release
// download addressed by the pinned version, not the GitHub API. It is a var,
// not a const, so a test can point it at a local server.
var ReleaseBase = "https://github.com/meop/sheesh/releases/download/"

// Assets are the pinned release's archives, keyed by the "<os>-<arch>" sheesh
// spells in their names: each one's sha256, and that of every file kept from
// it. `go run ./cmd/pin` prints an entry; see AGENTS.md for bumping the pin.
var Assets = map[string]pinned.Asset{
	"linux-x86_64":    {SHA256: "d100438a5f299429ac5871034f08c759fe192326e6f749fea436c421ff9b6cb4", Files: map[string]string{"kebab": "79e6898090f9ff2e37dc151792a65d5d0c008371fd18f333557b1fb43e86275d", "sheesh": "605ea52ce119cb09ab61dae6fb96bc2eefee2d792e9edb777f7cec5cabcfc933"}},
	"linux-aarch64":   {SHA256: "ac33af73ead23fe188317c6b23c33139d763f8d2eb828ac6a8a63cb856e8d814", Files: map[string]string{"kebab": "878ec3200351e8e94167fb944d287c8efe7767b1ea24b5b8a1b7fbd9e9228698", "sheesh": "87df8e040bf8b6f30eb4c2730d625937566bc693026bb18fe4e2a4b272c68def"}},
	"darwin-x86_64":   {SHA256: "9e7b1675a9f42a6bd2e758d42ed967ad5e2a19af46fb9ae30f151654c2bca59e", Files: map[string]string{"kebab": "ec031429616258fdad78b4776fa5c24b9d50e92ac1a27f2fd227bfdb5083134f", "sheesh": "d492899f1e6f92c54a2d2c686921b2a9255c8a20f46202597b16c3733b265506"}},
	"darwin-aarch64":  {SHA256: "ace638e8273bf9631ce0f68e4b292c753fd333adddaf315f16f0ddb653c7b54c", Files: map[string]string{"kebab": "8fa0cc33227b22bdc1d867d683a87ab01127dd7078fe49f24ba4bffe3483cdaf", "sheesh": "54e081f1af6beea4d2856e7ac8fd317868156bdc99af531f1db7b17c329be1c2"}},
	"windows-x86_64":  {SHA256: "e7eb6569cc90349f2ac0c0c1ed66066b414e012f4cbdd636bc22e48d09a36e09", Files: map[string]string{"kebab.exe": "ecaf6f3b0fb407302c917491278d4d69ede01a5e7e615cd88b6a24adf333c575", "sheesh.exe": "75675c1659e6665b00a2ccdb8ea2a7468ac1996fedd5bec2492912e8fc21e39b", "sheesh-gui.exe": "28133c8f9c5f87e2577cd2db4e4120d5c894891f9baa0dc2212f1247eff62077"}},
	"windows-aarch64": {SHA256: "1beff44469c6003a0a3f29336860dc4cabbb87d726bc4980023c56989f69fb7b", Files: map[string]string{"kebab.exe": "8aa6e0348b41ef72a41595da6b2b326fb04c8de8302205c66174cf8fc173d1de", "sheesh.exe": "ab4c12fd8a86d1e81cc0e16c84833be0bdcd821178a4619a290c9755e937a1db", "sheesh-gui.exe": "8273c1bd9425e7a748c7c2a408142533cfa5ba1daf54cb3f97480ca6bce1d525"}},
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
