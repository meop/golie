# golie

Go Library for Installing Executables. What [ghpm](https://github.com/meop/ghpm)
and [gotm](https://github.com/meop/gotm) share: the parts of a tool that
installs other tools, and keeps itself and its own helpers current.

```sh
go get github.com/meop/golie
```

## Packages

| Package | What it is |
|---------|------------|
| `fetch` | A download checked against a sha256 as it is written, and `.tar.gz` / `.zip` extraction that refuses an entry leaving its destination |
| `pinned` | Keeps a vendored helper at exactly the release a build pins, checked by content every time it is used |
| `sheesh` | The pinned [sheesh](https://github.com/meop/sheesh) release shims are stamped with |
| `shim` | Stamps and removes sheesh shims, and tells a tool's own from anything else on `PATH` |
| `upgrade` | A tool upgrading itself from its GitHub releases, over plain https |
| `vers` | Version strings, pins (`14`, `14.1`, `14.1.0`) and comparison |
| `ui` | One sink for output and prompts, with deferred blank lines |
| `lock` | The one-process-at-a-time lock over a tool's state directory |
| `home` | The user's home, with a guard that keeps tests out of the real one |

Nothing here calls the GitHub API or the `gh` CLI: every fetch is a plain
download of a URL the caller builds from a pinned version.

## Pinning a helper

A pin is a version plus the sha256 of the archive and of each file kept from
it. `cmd/pin` prints the entry for one archive:

```sh
go run github.com/meop/golie/cmd/pin -url https://github.com/meop/sheesh/releases/download/v0.1.1/sheesh-0.1.1-linux-x86_64.tar.gz kebab sheesh
```

## License

[MIT](LICENSE)
