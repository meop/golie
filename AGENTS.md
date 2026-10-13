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
again on Windows) → Release, which creates a GPG-signed tag `v<VERSION>`. The
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
- `sheesh/` — the pinned sheesh version and the digest of every platform's
  archive and files; `Sync` is `pinned.Ensure` for it
- `shim/` — `Stamper` stamps a shim with the vendored `kebab`, and only
  replaces or removes a file it can tell is one of its own
- `upgrade/` — `Self` reads the newest release from where `releases/latest`
  redirects, checks the archive against the release's `SHA256SUMS`, and
  replaces the running binary
- `vers/` — `SplitJunk` / `Normalize` for tags with leading junk, and
  `Constraint` for pins
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
  knows a tool's name or where its state lives

## Bumping sheesh

Change `sheesh.Version`, then replace every entry of `sheesh.Assets` with what
`cmd/pin` prints for that release's six archives (`kebab` and `sheesh`, and on
Windows `kebab.exe`, `sheesh.exe` and `sheesh-gui.exe`). Both tools move to it
when they take the new golie.
