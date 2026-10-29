# AGENTS.md

golie is the library ghpm and gotm share. A change here changes both, so a
package's behavior is documented where it lives and tested here, not in either
tool.

## After every code change

Run all four — do not skip any:

```sh
go build ./...
go test ./...
gofmt -w .
golangci-lint run ./...
```

## Comments

No comment explains why code does what it does where a unit test can show it: the test is the explanation, and it
fails when the reasoning stops holding. A comment is only for what no test can show.

## Keeping this file current

Update this file whenever a package is added, removed or renamed, or a
convention changes — do not wait to be asked. Keep descriptions conceptual:
what a package is for and what it guarantees, not function signatures. If you
notice a discrepancy between this file and the code, fix the file immediately.

Update `README.md` when what a tool built on golie can rely on changes: a
package added or removed, or a guarantee one makes.

## Publishing

To cut a release: bump the version in `VERSION` and push to `main`. The CI
pipeline (`.github/workflows/pipeline.yaml`) runs: Version check (skips the rest
if `v<VERSION>` already exists) → Validate (fmt + lint + test, and the tests
again on Windows and macOS) → Release, which creates a GPG-signed tag `v<VERSION>`. The
tag is the release: a Go module needs nothing else. Requires the
`GPG_PRIVATE_KEY` and `GPG_PASSPHRASE` repository secrets. Never create the tag
by hand.

A tool picks the release up with `go get github.com/meop/golie@v<VERSION>`.

## Project structure

- `fetch/` — `Download` hashes as it writes and removes a file that does not
  match; `Extract` unpacks `.tar.gz` and `.zip` with a strip count, and fails
  on an entry that would leave the destination or is not a plain file or
  directory. Standard library only
- `pinned/` — `Ensure` makes a directory hold a pinned tool's files, each with
  its pinned sha256, and nothing else beside what `Options.Keep` names. `Read`
  is the other direction, for `cmd/pin`
- `wedge/` — the pinned wedge version and the digest of every platform's
  archive and files; `Sync` is `pinned.Ensure` for it
- `shim/` — `Stamper` stamps a shim with the vendored `mallet`, and only
  replaces or removes a file it can tell is one of its own: one whose wedge
  trailer names a target under `Owned` (`StampedTarget` reads it). The trailer
  is wedge's format, never a string in its code, so a reworded message or a
  rename does not lose a tool its shims
- `upgrade/` — `Self` reads the newest release from where `releases/latest`
  redirects, checks the archive against the release's `SHA256SUMS`, and
  replaces the running binary. A tool that names `Dir`, where its installer
  puts it, upgrades only the copy there: one that runs from anywhere else is
  refused (`Check`, a `NotSelfInstalledError` the tool says in its own words)
  before anything is fetched, since whatever put it there is what upgrades it
- `vers/` — `SplitJunk` / `Normalize` for tags with leading junk, and
  `Constraint` for pins
- `manifest/` — the record of what a tool installed, and the one file both
  tools keep: `repo` (package name → source) and `extract` (key → pin level,
  version, assets, and the shim → binary and font maps). Also what a key is
  (`fzf`, `fzf@14`), the pin level a pin holds an install to, and what a
  pinned install's shim is called. Its JSON field names are a contract: a
  change to one is a change to every `~/.ghpm` and `~/.gotm` in use
- `layout/` — the directories of a state directory: `bin`, `vendor`,
  `extract/<key>/<version>` and the `extract/<key>/version` file shims are
  stamped through, `download/<source>/<version>`, the manifest and
  the lock
- `journal/` — what makes an install all or nothing: each mutation recorded
  with its undo before it is made, rolled back newest first when the op fails
  short of the manifest write. `Place` puts an unpacked version at its dir,
  beside one already there rather than over it; `Point` moves the version file
- `pointer/` — the version file: a regular file naming, relative to its own,
  the version dir, which a wedge shim stamped through it reads as that dir. The
  same on every platform, and read the same by every process, elevated or not,
  where Windows hides from an elevated process a link a user made
- `tidy/` — finds what the manifest does not account for (shims, extract
  dirs, cached downloads) and what it claims that is gone (broken installs),
  and removes either. No output and no prompts: the tool shows what a `Find`
  returned and asks
- `ui/` — deferred separators; see the comments in `ui.go`
- `lock/`, `home/` — small, and what their package comments say
- `cmd/pin/` — prints a `pinned.Asset` entry for an archive URL

## Conventions

- No GitHub API and no `gh` anywhere. A fetch is a plain https download of a
  URL built from a pinned version, never a "latest", with one exception that
  is the point of it: `upgrade` asks which release is newest
- Nothing fetched is used before it is checked: an archive against its pinned
  or published sha256, and a pinned tool's files against their own
- **The check on a vendored tool is its contents, not its version.** A sha256
  per file every time the tool is about to be used: cheaper than spawning it to
  ask its version, and it catches a copy that was damaged or swapped, which a
  version string does not. `.pin` in the directory records the version last
  synced only so a wrong copy of *this* version can be told from an ordinary
  copy of another, and said (`Options.Replacing`); it is never trusted
- No file or directory at the module root may differ from another only by
  case. Go refuses a module that has one — 0.1.0 shipped a `version` package
  beside the `VERSION` file and could not be fetched at all, which is why the
  package is `vers`
- A package takes the paths and names it needs as arguments. Nothing here
  knows a tool's name or where its state lives, nor what a package is: gotm's
  only one is `go`, and it passes that where ghpm passes a name
- The backend is here and the conversation is not. What a tool prints, asks
  and names its commands stays in the tool; `manifest`, `layout`, `journal`
  and `tidy` never print or prompt

## Bumping wedge

Change `wedge.Version`, then replace every entry of `wedge.Assets` with what
`cmd/pin` prints for that release's six archives (`mallet` and `wedge`, and on
Windows `mallet.exe`, `wedge.exe` and `wedge-gui.exe`). Both tools move to it
when they take the new golie.
