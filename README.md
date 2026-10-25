# ihatt — It Happens At That Time

A CLI/TUI temporal knowledge base over your local git repositories.

`ihatt` indexes the commits (and optionally GitHub or GitLab issues / PRs / MRs) of every git
repo it knows about into a single local database, then lets you ask questions
across all of them at once:

- *What was I working on last Tuesday?*
- *What changed across all my projects this week?*
- *Show me everything that happened on March 5th.*
- *Where did repo A reference a PR in repo B?*

Everything stays on your machine: data lives in a single
[bbolt](https://github.com/etcd-io/bbolt) file under `$XDG_DATA_HOME/ihatt`.

## Installation

### Homebrew

```bash
brew install rlespinasse/tap/ihatt
```

#### macOS Gatekeeper notice

On macOS, you may see a warning: _"Apple is not able to verify that it is free from malware that could harm your Mac or compromise your privacy."_

This is because the binary is ad-hoc signed but not notarized by Apple.

> [!NOTE]
> Apple notarization requires a paid Apple Developer account ($99/year). It consists of an automated malware scan — not a manual security review or code audit — and does not guarantee the software is safe. Open-source projects can be verified by reviewing the source code and build pipeline directly.

To allow it to run, either:

- **Via System Settings (UI):** Go to **System Settings > Privacy & Security**, scroll down, and click **Open Anyway** next to the blocked app message.
- **Via terminal:**

  ```bash
  xattr -d com.apple.quarantine $(which ihatt)
  ```

### From source

```bash
# from source (Go 1.24+)
go install github.com/rlespinasse/ihatt@latest

# or via the included justfile
just install        # go install with version baked in
just build          # produces ./bin/ihatt
```

### Binary releases

Download pre-built binaries from the [Releases](https://github.com/rlespinasse/ihatt/releases) page. Available for Linux, macOS, and Windows (amd64/arm64).

GitHub sync uses the [`gh`](https://cli.github.com/) credentials, so make sure
you've run `gh auth login` if you plan to use `ihatt github sync`.

GitLab sync works the same way with the [`glab`](https://gitlab.com/gitlab-org/cli) CLI:
run `glab auth login` before `ihatt gitlab sync`.

Self-hosted instances (GitHub Enterprise, self-hosted GitLab) are listed under `forges` in
the config file (see [Configuration](#configuration)). Log in to each one separately, for example
`gh auth login --hostname github.acme.io` or `glab auth login --hostname git.acme.io`.

### What forge sync stores

Both `github sync` and `gitlab sync` store, per tracked repo:

- **Issues** and **PRs / MRs**: title, state, author, assignees, milestone, labels, draft,
  comment count, dates, URL. GitHub PRs also get `reviewDecision`.
- **CI runs**: GitHub workflow runs or GitLab pipelines, keyed by commit. Shown as `[ci success]`
  or `[ci failure]` next to commits.
- **Comments** (opt-in with `--with-comments`): issue and PR/MR conversation comments, searchable
  with `ihatt search`. GitHub review comments and GitLab system notes are skipped.

Each sync is incremental after the first one: only items updated since the last sync are fetched.
Use `--full` to refetch everything. Deleted or transferred items are not removed.

Limits: GitLab review and approval state is not stored. Releases come from local git tags, not from
the forge. Tags are read when a repo is indexed, so re-run `ihatt repo add <path>` on repos tracked
before this change.

A commit that is the merge or squash commit of a PR/MR is labelled `[PR #N]` or `[MR !N]` in
`range`, `today`, and similar output. Rebase merges may not be linked.

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

Time-window commands (`today`, `yesterday`, `week`, `at`, `range`) accept filters that apply to
issues and PRs/MRs only: `--provider github|gitlab`, `--state open|closed|merged`, `--type issue|pr`,
`--label NAME`. Commits are never filtered.

## Commands

Commands that act on tracked repos (`sync`, `github sync`, `gitlab sync`, `xref`, `search`)
follow the working directory: run them inside a tracked repo and only that repo is used. Pass
`--repo <name>` to pick another one, or `--all` to act on every tracked repo.

| Command | Purpose |
| --- | --- |
| `ihatt init` | Create config dir, data dir, and an empty database. |
| `ihatt scan [--root PATH]... [--depth N]` | Discover git repos under one or more roots (recursively, unlimited depth by default; `--depth N` to limit) and index commits. Roots are remembered in config. |
| `ihatt repo add <path>` | Track a single repository. |
| `ihatt repo remove <name\|path>` | Stop tracking a repository. |
| `ihatt repo list` | Show tracked repos with commit counts and last-scan time. |
| `ihatt today` / `yesterday` / `week` | Pre-baked time-window queries. |
| `ihatt at <date>` | Activity on a specific date — accepts `2024-03-15`, `march 5`, etc. |
| `ihatt range <from> <to>` | Activity within an inclusive time range. |
| `ihatt search <query> [--author X] [--repo R] [--since D] [--until D]` | Search commit messages, authors, and files, plus synced issue/PR/MR comments. |
| `ihatt sync [--repo R] [--all] [--full] [--with-comments]` | Index commits, then sync GitHub and GitLab data, for the repos in scope. |
| `ihatt github sync [--repo R] [--full] [--with-comments]` | Pull issues, PRs, CI runs for tracked GitHub repos on github.com or a configured host (uses `gh`). |
| `ihatt gitlab sync [--repo R] [--full] [--with-comments]` | Pull issues, MRs, pipelines for tracked GitLab repos on gitlab.com or a configured host (uses `glab`). |
| `ihatt xref [--repo R]` | Build the cross-reference graph between tracked repos (commits referencing issues, PRs, MRs `!N` or SHAs). |
| `ihatt links <repo>` | Show inbound and outbound cross-references for one repo. |
| `ihatt tui` | Launch the interactive Bubble Tea UI. |

Run `ihatt <command> --help` for the full flag list of any command.

## TUI keys

| Key | Action |
| --- | --- |
| `j` / `k` (or arrows) | Move down / up |
| `enter` | Select item |
| `esc` / `backspace` | Back |
| `n` / `p` | Next / previous issue or PR in the repo view |
| `o` | Open the selected issue or PR in the browser |
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
forges:
  - host: github.acme.io
    kind: github
  - host: git.acme.io
    kind: gitlab
  - host: ssh.gitlab.acme.io      # remote host (SSH)
    kind: gitlab
    api_host: gitlab.acme.io      # host that serves the API, when different
```

`scan_roots` is also populated automatically the first time you run
`ihatt scan --root <path>`.

`forges` maps a host to its kind: `github`, `gitlab` or `unsupported`. The kind is detected from
the hostname: a host containing `github` is GitHub, one containing `gitlab` is GitLab (so
`github.com`, `github.acme.io` and `git.gitlab.acme.io` need no entry).

Set `api_host` when the remote host does not serve the API, which is common when remotes use
an SSH host such as `ssh.<host>`. Login is then done on `api_host`, not on the remote host.

For any other host, the sync commands ask on an interactive terminal whether it is `github`,
`gitlab` or `unsupported`, and save the answer under `forges`. Without a terminal (cron, CI),
those repos are skipped with a message naming the host; add the host to `forges` to classify
it ahead of time. Hosts marked `unsupported` are skipped silently after that.

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
  forge/     Shared sync watermarks and paging (GitHub and GitLab)
  github/    GitHub sync (issues, PRs, CI, comments) via gh
  gitlab/    GitLab sync (issues, MRs, pipelines, notes) via glab
  model/     Shared domain types
  query/     Time-window queries + result formatting
  store/     bbolt-backed persistence
  tui/       Bubble Tea views (dashboard / timeline / search / xref)
  xref/      Cross-reference extraction
```

## License

See [LICENSE](LICENSE).
