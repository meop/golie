# AGENTS.md

golie is the library ghpm and goup share. A change here changes both, so a
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
if `v<VERSION>` already exists) → Validate (fmt + lint) and Test (on every platform
and architecture ghpm and goup ship for, each on its own runner) → Release, which creates a GPG-signed tag `v<VERSION>`. The
tag is the release: a Go module needs nothing else. Requires the
`GPG_PRIVATE_KEY` and `GPG_PASSPHRASE` repository secrets. Never create the tag
by hand.

A tool picks the release up with `go get github.com/meop/golie@v<VERSION>`.

## Project structure

- `fetch/` — `Download` hashes as it writes and removes a file that does not
  match; `Extract` unpacks `.tar.gz` and `.zip` with a strip count, and fails
  on an entry that would leave the destination or is not a plain file or
  directory. Standard library only
- `vendored/` — a `Release` is where a vendored tool's releases are and how
  they are packed. `Ensure` installs the latest only where the directory holds
  none, and otherwise asks nothing of the network; `Refresh` is the one call
  that looks for a newer release, for a command the user runs to mean that.
  Either installs only what passed its checks — the archive against the
  release's manifest, the manifest against its signature when the tool names a
  key — and leaves the directory holding the release's files, its `.version`
  written last, and nothing else beside what `Options.Keep` names. `Label` is
  how a vendored tool is named wherever it is printed (`vendor - gh`), and
  `Said` how a refresh's outcome is, so every tool that vendors one reports it
  in the same words: a tool prints those, never its own wording
- `doctor/` — what a doctor command prints, the same in every tool: the title
  and rule, a `[PASS]`/`[WARN]`/`[FAIL]` line a check, `Present` for what was
  found (`present [version] — path`), `Vendored` for a vendored tool, labelled
  and versioned as its refresh says it, and `Size` for a cache. A tool's own
  checks go through these too, never a line of its own shape
- `wedge/` — wedge's `vendored.Release` for a platform: its archive, the files
  kept, and the release key
- `sig/` — `Verify` checks a detached OpenPGP signature; `ReleaseKey` is the key
  every meop release pipeline signs its `SHA256SUMS` with, built in and never
  fetched. `internal/testsign` makes a throwaway key for tests
- `shim/` — `Stamper` stamps a shim with the vendored `mallet`, and only
  replaces or removes a file it can tell is one of its own: one whose wedge
  trailer names a target under `Owned` (`StampedTarget` reads it). The trailer
  is wedge's format, never a string in its code, so a reworded message or a
  rename does not lose a tool its shims
- `upgrade/` — `Self` reads the newest release from where `releases/latest`
  redirects, checks the archive against the release's `SHA256SUMS` (and that
  against `SHA256SUMS.sig` when it names a `Key`), and
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
  change to one is a change to every `~/.ghpm` and `~/.goup` in use
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
  returned and asks. Inside an install it keeps the version file as a file, not by
  name: where names fold case (macOS, Windows) it can sit under a name an older layout left, as Go's `VERSION`.
  A version a running program was started from is never removed: a prune keeps it with an `InUse`, and a tidy does
  not list it until the program has stopped. Nor is a shim a program runs from, as Windows keeps one a stamp set
  aside (`running.Held`)
- `running/` — the programs running, by the path each was started from: `/proc` on Linux, `kern.procargs2` on
  macOS, a process snapshot on Windows; and `Held`, a file Windows will not delete because a program runs from it,
  which it names by the path it started from rather than the one it has now. A long-running server outlives the upgrade that replaced it, and its old
  binary has to stay: macOS's firewall drops connections to a program it can no longer read, and Windows will not
  delete a running program's file
- `ui/` — deferred separators; see the comments in `ui.go`, and **Line
  spacing** below
- `lock/`, `home/` — small, and what their package comments say

## Line spacing

A blank line means the run has moved to the next step of its flow. Nothing else gets one.

- Exactly one blank line before every prompt: never none, never two.
- A prompt, the answer typed to it, and the output that answer lets through are one step: the output starts on the line
  right below the answer.
- Lines of one kind in a row (downloads, bins found, refresh lines, a script's output) run tight: no blank between them,
  none after them.
- A step never ends itself with a blank line; the step after it asks for one.
- `-y` answers a prompt without showing it; the prompt's blank line still comes, where the prompt would have been.

The rule is the same, word for word, in golie, ghpm, goup, shire, wut and wut-config.

Here: `ui` is where every tool built on golie gets its blank lines. `Break` asks for one, deferred, so a run never starts or ends on one and two never stack. `Prompt`, `Menu`, `Confirm` and `Table` ask for their own before them and none after, so a menu shown again inside one prompt (a "show more") gets its blank too. A tool's `--yes` asks for the confirm's blank itself. Nothing in golie prints a blank outside `ui`.

## Testing on a real host

CI runs the tests on Linux and Windows on x86_64 and arm64, and macOS on arm64. To run them on a machine of your own over ssh:

- Windows: build the test binary anywhere (`GOOS=windows go test -c -o <pkg>.test.exe ./<pkg>/`), copy it over and run
  it there. In PowerShell, quote a test flag whole (`'-test.v=true'`): PowerShell splits a bare `-test.v` at its dot.
- macOS and Windows fold the case of file names: a test that makes two names differing only by case (or a link from
  one to the other) cannot run there, so test the behavior the folding causes instead.
- In a Windows ssh login, OpenSSH's own tools (`ssh`, `ssh-keygen`) can lose their output: run what uses them from the
  desktop session.

## Conventions

- No GitHub API and no `gh` anywhere. Which release is newest is read from
  where a release page's `latest` redirects, and every fetch is a plain https
  download
- **Nothing is pinned.** A vendored tool and a tool's own upgrade both take the
  latest release, so a fix in one reaches the others without a golie release.
  What a pin used to guarantee comes from the release instead: nothing fetched
  is used before its archive matches the release's checksum manifest, and that
  manifest its signature, where the release is signed
- **Nothing goes online unasked.** A vendored tool is fetched where it is
  missing, and otherwise moves only when the user runs the tool's `refresh`
- No file or directory at the module root may differ from another only by
  case. Go refuses a module that has one — 0.1.0 shipped a `version` package
  beside the `VERSION` file and could not be fetched at all, which is why the
  package is `vers`
- A package takes the paths and names it needs as arguments. Nothing here
  knows a tool's name or where its state lives, nor what a package is: goup's
  only one is `go`, and it passes that where ghpm passes a name
- The backend is here and the conversation is not. What a tool prints, asks
  and names its commands stays in the tool; `manifest`, `layout`, `journal`
  and `tidy` never print or prompt

## A new wedge release

Nothing to do here: a tool's `refresh` takes it. A new platform, or a change to
how wedge names its archives or what is in them, is a change to `wedge.Release`.
