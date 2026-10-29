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
const Version = "0.5.1"

// ReleaseBase is where wedge's releases are downloaded from: a plain release
// download addressed by the pinned version, not the GitHub API. It is a var,
// not a const, so a test can point it at a local server.
var ReleaseBase = "https://github.com/meop/wedge/releases/download/"

// Assets are the pinned release's archives, keyed by the "<os>-<arch>" wedge
// spells in their names: each one's sha256, and that of every file kept from
// it. `go run ./cmd/pin` prints an entry; see AGENTS.md for bumping the pin.
var Assets = map[string]pinned.Asset{
	"linux-x86_64":    {SHA256: "6c4d7691546f1e9ec9fbc1e8bb4d0f30a9f25570fdbbe3192d8fbbd6918321bf", Files: map[string]string{"mallet": "a1a270ec3d7983eef9de01544a1f3a644f22a34881cbaa7120fe6a3673ae2d4c", "wedge": "034fbc90cb790b6bad82775e887e53288c37dd9a7ecbd217116a1c6132ad2527"}},
	"linux-aarch64":   {SHA256: "ee03b249f22e4a18ca2452d8f6fbcaeb1a4a66bd3dfeab38d05577b23374f597", Files: map[string]string{"mallet": "8ae8a433708526f1adc64a5cbbd299d32ea245111cab46bc199919ee78f4ee34", "wedge": "9c82f7749f438fae698d2c9cc4beff5d12c82a55f05cb50c1df546536814f0c1"}},
	"darwin-x86_64":   {SHA256: "708900c4182b508eb7a9eb80df43d4cf30c19e21afc23e3f8437afcea8cbc6c9", Files: map[string]string{"mallet": "4d963c02caea7b7e547ffa4ae009d97373004415074f44f5f381e21ffad56c69", "wedge": "cc6036dfca62cd746a38c74f527b85ea891b5f1d2f55454b5883884610724dca"}},
	"darwin-aarch64":  {SHA256: "2de91d9d97561de3c6b24381aeb9b7a8a46527bbffdd87c4c5c78ff0426188bc", Files: map[string]string{"mallet": "b1c9118323891f7d71cdbd40abfbdeac3a442e2b9b4b07c69d3103a2585e3e44", "wedge": "d07f41aefd795abdd7beff4317e8311fecd537c50b81eb698190c82d855b4faf"}},
	"windows-x86_64":  {SHA256: "26adcf4723d7ff51a74eef1b12ad80aae76e79d5be4f3e431fa6334590f3e92f", Files: map[string]string{"mallet.exe": "62f1cae9176a73933befd9883c68ecb2ac28baaf408746e3a0d3d6ba4bddb7b3", "wedge.exe": "c60e2576c47befb6ee54b874482fab56bad91554f760afe9eadca9c806c93d83", "wedge-gui.exe": "4fc3076f7b630c3da5e34a0b37f208bfd0326ac35ea79a6c9b3988f5286f2390"}},
	"windows-aarch64": {SHA256: "8dc54a6aba9174ddfd2bade727d78fb8737c93ed718d4293f2393f97de27a852", Files: map[string]string{"mallet.exe": "ebd8e4b38b1d1a5329ce52b7bc6d6e1c98b5cf1544fc8f4c28454a67096f404e", "wedge.exe": "1978626200f479989ebb9e61fdc826e375ae70f5dbbf2b6f2cdbc7271c5e73d3", "wedge-gui.exe": "249b1d3c9eb7a5568125e0885d1229b4c470720422ff581372d05f2f6d9da863"}},
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
