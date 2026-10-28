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
const Version = "0.3.0"

// ReleaseBase is where wedge's releases are downloaded from: a plain release
// download addressed by the pinned version, not the GitHub API. It is a var,
// not a const, so a test can point it at a local server.
var ReleaseBase = "https://github.com/meop/wedge/releases/download/"

// Assets are the pinned release's archives, keyed by the "<os>-<arch>" wedge
// spells in their names: each one's sha256, and that of every file kept from
// it. `go run ./cmd/pin` prints an entry; see AGENTS.md for bumping the pin.
var Assets = map[string]pinned.Asset{
	"linux-x86_64":    {SHA256: "7194eae365b17b9fd6e86d610918fb63df1c38576b956d11644cdd39d7e6a9d8", Files: map[string]string{"mallet": "561038335c311f60f8db5f79982707fb8574b9e013be79c709a0d45d67378921", "wedge": "37bc8959949504db118d84cb14d4ce77b0fd0bceb86d9a9c1435d743dc9c94ad"}},
	"linux-aarch64":   {SHA256: "63cf702bebdc36df94077f503e76128a60e0c58df5b782669ba428b31eabd814", Files: map[string]string{"mallet": "0d92dbab7c33b3153a5554f5a20eb1167dd014e28efab1181dad8398879bb611", "wedge": "ed69092138be3876daa76d7a26f477afae39d50658dbb822b508bb999d6df2bd"}},
	"darwin-x86_64":   {SHA256: "b8df892e0a5cf8fab7dcc9cb39fd0548aa6f11fbcd91859363c891558220b9ce", Files: map[string]string{"mallet": "785f38a0c609357624b81fdf6d53d053574a7f383a8c21f1a800f34ad3f9ebb4", "wedge": "2aa3dfa73018d8ba0a1a6af6123087b236f9b3d310512f1c4c58ef133766e8a1"}},
	"darwin-aarch64":  {SHA256: "a2656a2d950ab10f9aa5bc2d4400149e31e7d9d6e378843b9e19617b412d4dac", Files: map[string]string{"mallet": "d077915c3b078361f9755eb7ce6e548ec7c988a05f7ccddebd66ba9c7e54c46d", "wedge": "b4a3de0db569cfb0ea30775d6c99a3ff23d895457be15285f010fe388a488061"}},
	"windows-x86_64":  {SHA256: "1be785775b1295150ad17f1c23aacbe9410ca3c9e4ac231995411e62b29e7b51", Files: map[string]string{"mallet.exe": "300c78673bc62d33327a8aa606da1ad68fc96ec2c3a606723bec39ac65787f66", "wedge.exe": "9cb5e705bed68d9f35ec901c9f91681e5d594759fa883ebb3b552dba316fcc1b", "wedge-gui.exe": "5b334f9ab63ed065f06a51402e501471080d2217269d6eef3418fa96e7f78bf8"}},
	"windows-aarch64": {SHA256: "4c9c8dfa4b68640bd68f5fbc00f112d0be8683665508b7fa450546326a611821", Files: map[string]string{"mallet.exe": "46c4985297fa4eab0b4abe99d829a3caaf0857f8eb615d9363940bc285ed5d50", "wedge.exe": "180dce9d3f00a932a5f46a7c404f2a9824e369ecc6376603893dd3257fdd93ad", "wedge-gui.exe": "ab6b1912190e0d830a8c5e8b1b42bc0368cadea21cc7cbc584e5e1e0a309cb7f"}},
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
