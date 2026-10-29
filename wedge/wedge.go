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
const Version = "0.4.0"

// ReleaseBase is where wedge's releases are downloaded from: a plain release
// download addressed by the pinned version, not the GitHub API. It is a var,
// not a const, so a test can point it at a local server.
var ReleaseBase = "https://github.com/meop/wedge/releases/download/"

// Assets are the pinned release's archives, keyed by the "<os>-<arch>" wedge
// spells in their names: each one's sha256, and that of every file kept from
// it. `go run ./cmd/pin` prints an entry; see AGENTS.md for bumping the pin.
var Assets = map[string]pinned.Asset{
	"linux-x86_64":    {SHA256: "289cc5c9dcbc0f7d05a86f3e1e6c4f9f62d7e1d1cf597e93484f410c19260892", Files: map[string]string{"mallet": "715ffac9e40d8657fe07d165213aa8736b12709ca91cceb557d1d3780b1070c4", "wedge": "4ca0e2b8263bec3e9f882b369323c5e74589648c9fc9e27edea93984deb24b0e"}},
	"linux-aarch64":   {SHA256: "b245495599be1d008608371e1c711b9e2c300f28f4e6a20fb91c0845a87fed5c", Files: map[string]string{"mallet": "adf7ec9fa1b90a3b7594817ab66bcca2f1b07ae3ce38bd8c59758e8f525a79a8", "wedge": "731b681bf073e798088d88830d455f825f437d11b09d7d2842330d269fadc237"}},
	"darwin-x86_64":   {SHA256: "6aae0572945c09fab368b180df6aca80bc6f4600c4ae652bd760f087a3cb0e6e", Files: map[string]string{"mallet": "adcf7b4864e55d4f71b48042217b443a2245adb8f3bdbca9f9014bfa7786f12b", "wedge": "c823a85d574eb9259277fd930519b9474a6a6acc17ab8d5f381fc60f95364f83"}},
	"darwin-aarch64":  {SHA256: "4f0de74cb61bf872547b588500e5974133a0a2dfd0fe70ee7b2d3db27651f6df", Files: map[string]string{"mallet": "06b5b177097ce60ab478f545936ad503787ed9229488d41520db5d777916c9a1", "wedge": "6508e6ce567279685f10843380886a36892da14543a40cd147d7468b2a98db1e"}},
	"windows-x86_64":  {SHA256: "df6871d5ab5316663d98b6a1e3f9887523e2d0e7ba9de48f5252feba284e0661", Files: map[string]string{"mallet.exe": "f7a1c4241002446d010512018d819c84c2f952d718ba0a88fd47d4cbaa3592b4", "wedge.exe": "01dee256df67f531fe9b919ab2992208b59fae420d7b2b880573d01cc1e7b7f4", "wedge-gui.exe": "5611179cfb102f14f6cd974db8d8ec1cacc2f12368ef36939abd6835f6fd0923"}},
	"windows-aarch64": {SHA256: "645d93243c5dacd33f7f5b02a46c05eac8fe7f534f36afb1f6640a26def7499c", Files: map[string]string{"mallet.exe": "33f86a4065184c7d53b0d1ddb6e453b871904ae32093441f812ab8432f2b359a", "wedge.exe": "a36d9c6e587edc4fb38b4cc819c98cfff2e42ec514ecff71a7d75b8141f34df0", "wedge-gui.exe": "15be22629dae8fc1bdcbeeba42be650356327a9b374cbc0d7856ed6e41f28505"}},
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
