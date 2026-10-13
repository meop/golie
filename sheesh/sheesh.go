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
const Version = "0.1.1"

// ReleaseBase is where sheesh's releases are downloaded from: a plain release
// download addressed by the pinned version, not the GitHub API. It is a var,
// not a const, so a test can point it at a local server.
var ReleaseBase = "https://github.com/meop/sheesh/releases/download/"

// Assets are the pinned release's archives, keyed by the "<os>-<arch>" sheesh
// spells in their names: each one's sha256, and that of every file kept from
// it. `go run ./cmd/pin` prints an entry; see AGENTS.md for bumping the pin.
var Assets = map[string]pinned.Asset{
	"linux-x86_64":    {SHA256: "c1a08126ebb3117dd0034ec1ced849c6ef6df3a214f7b4884f07f34ad754de37", Files: map[string]string{"kebab": "1088d3116d0b6827c1d62d3a4dfcd5c1bc85624f535b507ea0d0e7aec30fc578", "sheesh": "3083131e230c9a7488a32492cd7c0cb845f290369744e9fe4255770ca06b659b"}},
	"linux-aarch64":   {SHA256: "1552842af9e910f56fa45c7b6197086b784c3db1826cc3e8b4e1648d3ce885e9", Files: map[string]string{"kebab": "dd7bf5a2f4a7ba7b43d651b81e5263c0bb2640659ed4a080bcd7da64455b8c46", "sheesh": "7626255a96786561a090532d6ca70633f2dfd7ac75824caf182cfce0cad5db5e"}},
	"darwin-x86_64":   {SHA256: "8d5f0d09974f2208195c26bd8960ef0c10bc8207f617bea3a95ccb2cf98ce8bc", Files: map[string]string{"kebab": "5438270d53ab96dcc455011f7b75625b0a3de23bb22c83cf5b24fe221debc818", "sheesh": "11d6fea1822888bfca6c34c6b0e3774f3be9688806b1542be6ce78bb79f521f4"}},
	"darwin-aarch64":  {SHA256: "3c697949bcdaa32707adb06e803a3875f41b61e191e0e8beba7a4e62c204b87b", Files: map[string]string{"kebab": "04c419c78e8326b5c9897060c9ebfd0830c754c52a7e455f8148d03255c54f82", "sheesh": "c224882b4301b9296946164b7fc1045cdc3534b1807a046b5c5ebabec1065558"}},
	"windows-x86_64":  {SHA256: "d3192a5661090a28278a784f0988df8eaef19c5be96f761b8fe8a7b26f4a3155", Files: map[string]string{"kebab.exe": "a42b54f3e8ebdbbd4df67f816973a89a7bd1b9e100228e70d84bc96a38f06dbd", "sheesh.exe": "4a4bd72c077b1e978f9d1ed69de437bb6ab40df2d7d33cbe41245b1be2f8ba82", "sheesh-gui.exe": "c0c6c39e61d38cac8eb3196aa6d29b4ce1d2e7872d3665edae13da815f0a33bf"}},
	"windows-aarch64": {SHA256: "9f6ffa11fce29e3ce5a2d796123af7af003a7aad22d8ec97a14263c811d6cd66", Files: map[string]string{"kebab.exe": "bd51d7c51bad273a31f48703576c704b4fbea208f328fb34406e1c4b5e78cb8c", "sheesh.exe": "2850ad5143c9ef2eda39686d6201a19c933b5aca437ade0a9d6f61028891f9a9", "sheesh-gui.exe": "f24524426b478a747f7f54ce640e34955302313e372d866257281e1a2859efc8"}},
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
