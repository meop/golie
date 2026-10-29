# golie

Go Library for Installing Executables. What [ghpm](https://github.com/meop/ghpm)
and [gotm](https://github.com/meop/gotm) share: the parts of a tool that
installs other tools, and keeps itself and its own helpers current.

## Install

```sh
go get github.com/meop/golie
```

## Usage

| Package | What it is |
|---------|------------|
| `fetch` | A download checked against a sha256 as it is written, and `.tar.gz` / `.zip` extraction that refuses an entry leaving its destination |
| `pinned` | Keeps a vendored helper at exactly the release a build pins, checked by content every time it is used |
| `wedge` | The pinned [wedge](https://github.com/meop/wedge) release shims are stamped with |
| `shim` | Stamps and removes wedge shims, and tells a tool's own from anything else on `PATH` |
| `upgrade` | A tool upgrading itself from its GitHub releases, over plain https, and only the copy its own installer placed |
| `vers` | Version strings, pins (`14`, `14.1`, `14.1.0`) and comparison |
| `manifest` | The record of what is installed: one JSON file, keyed `name` or `name@pin`, the same for every tool |
| `layout` | Where a tool's state lives: `bin`, `vendor`, `extract/<key>`, `download/<source>/<version>` |
| `journal` | All-or-nothing installs: every step recorded with its undo, rolled back unless the manifest was written |
| `tidy` | Finds and removes broken installs and orphaned shims, extract dirs and downloads |
| `ui` | One sink for output and prompts, with deferred blank lines |
| `lock` | The one-process-at-a-time lock over a tool's state directory |
| `home` | The user's home, with a guard that keeps tests out of the real one |

Nothing here calls the GitHub API or the `gh` CLI: every fetch is a plain
download of a URL the caller builds from a pinned version.

## Pinning a helper

A pin is a version plus the sha256 of the archive and of each file kept from
it. `cmd/pin` prints the entry for one archive:

```sh
go run github.com/meop/golie/cmd/pin -url https://github.com/meop/wedge/releases/download/v0.3.0/wedge-0.3.0-linux-x86_64.tar.gz mallet wedge
```

## Verifying releases

A release is a GPG-signed tag. Verify it with the public release key:

```sh
git verify-tag v0.1.1
```

## License

[MIT](LICENSE)
