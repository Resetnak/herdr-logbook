# Herdr Memory

Herdr Memory is a local, Markdown-first working logbook for developers using Herdr. It keeps the active task, quick captures, technical decisions, and project notes close to the terminal without introducing a proprietary document format.

The core loop is intentionally small: open the Hub for the active repository, capture context before it disappears, preview Markdown, search registered projects, and continue full editing in your normal editor.

```text
┌ Scopes ─────────┬ Notes ─────────────────────┬ Preview ───────────────────────┐
│ ● Now           │ now.md                     │ # Now                         │
│ Project Inbox   │ auth-notes.md              │ ## Current task               │
│ Project Notes   │ use-redis.md               │ Implement token rotation.     │
│ Decisions       │                            │                               │
│ Global Inbox    │                            │ ## Next steps                  │
│ All Notes       │                            │ - [ ] Add replay detection.   │
└─────────────────┴────────────────────────────┴────────────────────────────────┘
 api-gateway · feature/token-rotation · central store · / search · ? help
```

## Workflows

- `c` and `C` capture into the project or global monthly inbox.
- `n` creates a project note; `d` creates a dated decision record.
- `/` searches the disposable index across registered project stores.
- `e` suspends the Hub and opens the selected Markdown file with the configured editor.
- `now.md` is scaffolded once and always appears first in the project view.

The Hub uses Bubble Tea and adapts from three panes at 110 columns to one active pane below 70 columns. Markdown preview is rendered in process with Glamour; an external `glow` binary is not required.

## Installation status

Release packaging is configured for Linux, macOS, and Windows on amd64 and arm64. The install scripts download the exact manifest version and verify `checksums.txt`. A public `v0.1.0` release must not be claimed until the manual compatibility matrix in [docs/herdr-compatibility.md](docs/herdr-compatibility.md) is complete.

For local development:

```bash
go test ./...
go build -o bin/herdr-memory ./cmd/herdr-memory
herdr plugin link "$(pwd)" --enabled
herdr plugin action invoke open --plugin herdr-memory
```

On Windows PowerShell:

```powershell
go test ./...
go build -o bin\herdr-memory.exe .\cmd\herdr-memory
herdr plugin link (Get-Location)
```

`herdr plugin link` does not run build commands. Build the binary first.

## Commands

```text
herdr-memory tui --view now|project|global|all [--project-root PATH]
herdr-memory capture --scope project|global [--text TEXT | --stdin | --selected]
herdr-memory decision [--title TEXT] [--project-root PATH]
herdr-memory init --storage central|repo [--project-root PATH]
herdr-memory doctor [--json] [--project-root PATH]
herdr-memory keybinds
herdr-memory paths [--json] [--project-root PATH]
herdr-memory index rebuild [--project-root PATH]
herdr-memory version
```

Without a capture source, `capture` opens the textarea UI. Without `--title`, `decision` prompts for a title. Decision creation opens the configured external editor unless `--no-edit` is used for automation.

## Storage and project resolution

Central storage is the default and leaves repositories untouched:

```text
$HERDR_PLUGIN_STATE_DIR/
├── store/projects/p_<sha256>/{now.md,inbox,notes,decisions}
├── store/global/{inbox,notes,decisions}
├── registry/projects.toml
└── cache/index-v1.json
```

`init --storage repo` explicitly opts into `.herdr/memory/` and prints a suggested ignore rule; it never edits `.gitignore`. Initialization preserves every existing non-empty scaffold file. Cache files are disposable and rebuild from Markdown with `index rebuild`.

Context priority is explicit `--project-root`, Herdr worktree path, focused pane cwd, workspace cwd, `--cwd`, then process cwd. Git worktrees share identity through a credential-free remote fingerprint or Git common directory. Non-Git projects use the canonical path. `.herdr-memory.toml` may select a monorepo subproject but cannot escape the repository through traversal or symlinks.

## Configuration

Configuration is optional and read from `$HERDR_PLUGIN_CONFIG_DIR/config.toml`; see [config.example.toml](config.example.toml). Editor precedence is `editor.command`, `$VISUAL`, `$EDITOR`, then platform defaults. Commands are executed as argv, never interpolated through a shell.

Unknown configuration keys produce warnings. Invalid types and values fail explicitly. User configuration is not rewritten during compatible migrations.

## Privacy and data safety

Normal runtime has no telemetry, network requests, cloud sync, AI calls, automatic Git operations, or execution of commands from notes. Selected terminal text never appears in diagnostics. Git credentials are removed before registry and diagnostic output. Captures use bounded locks, same-directory temporary files, filesystem sync, and atomic replacement.

The index reads only Markdown under known memory roots, skips hidden directories and symlinks, and self-heals after cache corruption. Removing the plugin does not remove Markdown data.

## Limits and roadmap

Herdr Memory is not a general knowledge-management system, collaboration service, sync engine, browser UI, or custom text editor. Archive/trash, saved searches, templates, backlinks, and optional external Glow integration are deferred until the core workflow has real usage evidence. AI, cloud sync, and graph visualization are explicit non-goals.

Screenshots or a terminal recording will be added only after the real-host release matrix passes. See [CHANGELOG.md](CHANGELOG.md), [SECURITY.md](SECURITY.md), and [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
