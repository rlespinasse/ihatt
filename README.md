# ihatt — It Happens At That Time

A CLI/TUI temporal knowledge base over your local git repositories.

`ihatt` indexes the commits (and optionally GitHub issues / PRs) of every git
repo it knows about into a single local database, then lets you ask questions
across all of them at once:

- *What was I working on last Tuesday?*
- *What changed across all my projects this week?*
- *Show me everything that happened on March 5th.*
- *Where did repo A reference a PR in repo B?*

Everything stays on your machine: data lives in a single
[bbolt](https://github.com/etcd-io/bbolt) file under `$XDG_DATA_HOME/ihatt`.

## Installation

```bash
# from source (Go 1.24+)
go install github.com/rlespinasse/ihatt@latest

# or via the included justfile
just install        # go install with version baked in
just build          # produces ./bin/ihatt
```

GitHub sync uses the [`gh`](https://cli.github.com/) credentials, so make sure
you've run `gh auth login` if you plan to use `ihatt github sync`.

## Quickstart

```bash
ihatt init                       # create config + database
ihatt scan --root ~/code         # discover and index every git repo under ~/code
ihatt today                      # what did I touch today?
ihatt at "march 5"               # what happened on March 5?
ihatt search "auth migration"    # full-text across commit messages
ihatt tui                        # interactive view
```

Subsequent `ihatt scan` runs are incremental — only new commits are indexed.

## Commands

| Command | Purpose |
| --- | --- |
| `ihatt init` | Create config dir, data dir, and an empty database. |
| `ihatt scan [--root PATH]... [--depth N]` | Discover git repos under one or more roots and index commits. Roots are remembered in config. |
| `ihatt repo add <path>` | Track a single repository. |
| `ihatt repo remove <name\|path>` | Stop tracking a repository. |
| `ihatt repo list` | Show tracked repos with commit counts and last-scan time. |
| `ihatt today` / `yesterday` / `week` | Pre-baked time-window queries. |
| `ihatt at <date>` | Activity on a specific date — accepts `2024-03-15`, `march 5`, etc. |
| `ihatt range <from> <to>` | Activity within an inclusive time range. |
| `ihatt search <query> [--author X] [--repo R] [--since D] [--until D]` | Search commit messages, authors, and files. |
| `ihatt github sync [--repo R]` | Pull issues and PRs for tracked GitHub repos (uses `gh`). |
| `ihatt xref [--repo R]` | Build the cross-reference graph between tracked repos (commits referencing other repos' issues/PRs/SHAs). |
| `ihatt links <repo>` | Show inbound and outbound cross-references for one repo. |
| `ihatt tui` | Launch the interactive Bubble Tea UI. |

Run `ihatt <command> --help` for the full flag list of any command.

## TUI keys

| Key | Action |
| --- | --- |
| `j` / `k` (or arrows) | Move down / up |
| `enter` | Select item |
| `esc` / `backspace` | Back |
| `tab` | Switch between panes |
| `/` or `s` | Search |
| `t` | Timeline view |
| `d` | Dashboard |
| `l` | Cross-references view |
| `r` | Refresh data |
| `?` | Toggle help |
| `q` / `ctrl-c` | Quit |

## Configuration

`ihatt` follows the XDG base-directory spec.

| Path | Default | Override |
| --- | --- | --- |
| Config file | `~/.config/ihatt/config.yaml` | `$XDG_CONFIG_HOME/ihatt/config.yaml` |
| Database | `~/.local/share/ihatt/ihatt.db` | `$XDG_DATA_HOME/ihatt/ihatt.db` |

`config.yaml` keys:

```yaml
scan_roots:
  - /Users/me/code
  - /Users/me/work
```

`scan_roots` is also populated automatically the first time you run
`ihatt scan --root <path>`.

Run `just paths` to print the resolved locations on your machine.

## Development

A `justfile` wraps the common workflows — run `just` for the full list.

| Recipe | What it does |
| --- | --- |
| `just build` | Build `./bin/ihatt` with the git-described version baked in. |
| `just install` | `go install` the same binary. |
| `just run -- <args>` | Run from source: `just run today`, `just run scan --root .`. |
| `just tui` | Run the TUI from source. |
| `just scan ROOTS="~/code ~/work"` | Convenience wrapper that expands each path into a `--root` flag. |
| `just today` / `yesterday` / `week` | Source-mode shortcuts. |
| `just gh-sync` / `just xref` | Source-mode wrappers. |
| `just fmt` / `just fmt-check` | Format or verify formatting. |
| `just vet` / `just test` | Static analysis and tests (race detector on). |
| `just check` | The CI gate: `fmt-check` + `vet` + `test`. |
| `just tidy` | `go mod tidy`. |
| `just release-snapshot` | `goreleaser build --snapshot --clean --single-target`. |
| `just completion zsh` | Print a shell completion script (bash/zsh/fish/powershell). |
| `just paths` | Print the resolved config + data paths. |
| `just db-reset` | Delete the local database (asks for confirmation). |
| `just clean` | Remove `bin/` and `dist/`. |

### Layout

```
cmd/         Cobra commands (one file per subcommand)
internal/
  config/    XDG paths + viper-backed config
  git/       Repo discovery + commit indexer (go-git)
  github/    GitHub sync (issues/PRs) via gh
  model/     Shared domain types
  query/     Time-window queries + result formatting
  store/     bbolt-backed persistence
  tui/       Bubble Tea views (dashboard / timeline / search / xref)
  xref/      Cross-reference extraction
```

## License

See [LICENSE](LICENSE).
