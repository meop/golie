# golie

Go Library for Installing Executables. What [ghpm](https://github.com/meop/ghpm)
and [goup](https://github.com/meop/goup) share: the parts of a tool that
installs other tools, and keeps itself and its own helpers current.

## Install

```sh
go get github.com/meop/golie
```

## Usage

| Package | What it is |
|---------|------------|
| `fetch` | A download checked against a sha256 as it is written, and `.tar.gz` / `.zip` extraction that refuses an entry leaving its destination |
| `vendored` | Keeps a helper a tool vendors at its latest release: installed where missing, moved only by an explicit refresh, every release checked against its checksums and signature first |
| `wedge` | The [wedge](https://github.com/meop/wedge) release shims are stamped with, as `vendored` keeps it |
| `sig` | Checks a release's checksum manifest against its detached signature, with the meop release key built in |
| `shim` | Stamps and removes wedge shims, and tells a tool's own from anything else on `PATH` |
| `upgrade` | A tool upgrading itself from its GitHub releases, over plain https and checked against their signature, and only the copy its own installer placed |
| `vers` | Version strings, pins (`14`, `14.1`, `14.1.0`) and comparison |
| `manifest` | The record of what is installed: one JSON file, keyed `name` or `name@pin`, the same for every tool |
| `layout` | Where a tool's state lives: `bin`, `vendor`, `extract/<key>/<version>` and its `version` file, `download/<source>/<version>` |
| `pointer` | The `version` file a shim is stamped through, the same on every platform |
| `journal` | All-or-nothing installs: every step recorded with its undo, rolled back unless the manifest was written |
| `tidy` | Finds and removes broken installs and orphaned shims, extract dirs and downloads |
| `ui` | One sink for output and prompts, with deferred blank lines |
| `lock` | The one-process-at-a-time lock over a tool's state directory |
| `home` | The user's home, with a guard that keeps tests out of the real one |

Nothing here calls the GitHub API or the `gh` CLI: the newest release is read
from where a release page's `latest` address redirects, and every fetch is a
plain download.

## Verifying releases

A release is a GPG-signed tag. Verify it with the public release key:

```sh
git verify-tag v0.1.1
```

## License

[MIT](LICENSE)
