<div align="center">

# 📓 Herdr Logbook

**Local, offline, Markdown-first working memory for developers using Herdr.**

Keep the active task, quick captures, technical decisions, and project notes right next to the terminal — no proprietary format, no cloud, no telemetry.

<br>

[![CI](https://github.com/Resetnak/herdr-logbook/actions/workflows/ci.yml/badge.svg)](https://github.com/Resetnak/herdr-logbook/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platforms](https://img.shields.io/badge/platforms-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)](docs/herdr-compatibility.md)
[![Herdr](https://img.shields.io/badge/Herdr-%E2%89%A50.7.0-2088FF)](herdr-plugin.toml)
[![Status](https://img.shields.io/badge/status-pre--release-orange)](CHANGELOG.md)

<br>

**English** · [Čeština](README.cs.md)

[Why](#why-im-building-this-plugin) · [Installation](#installation) · [Commands](#commands) · [Storage](#storage-and-project-resolution) · [Configuration](#configuration) · [Privacy](#privacy-and-data-safety) · [Contributing](CONTRIBUTING.md)

</div>

---

Herdr Logbook is a local, Markdown-first working logbook for developers using Herdr. It keeps the active task, quick captures, technical decisions, and project notes close to the terminal without introducing a proprietary document format.

The core loop is intentionally small: open the Hub for the active repository, capture context before it disappears, preview Markdown, search registered projects, and continue full editing in your normal editor.

## Why I’m building this plugin

The useful context around a coding task often outlives the terminal session that produced it, but it does not need another cloud service or proprietary database. I’m building Herdr Logbook to keep that context one shortcut away inside Herdr: local, searchable, and stored as Markdown that remains useful without the plugin.

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

## Highlights

- **Markdown stays canonical.** Every note is a plain `.md` file you own. The search cache is disposable and rebuilds itself.
- **Project-aware.** Notes are scoped to the repository you are in, resolved through Herdr context, Git worktrees, or the current directory.
- **Offline and private.** No telemetry, network calls, cloud sync, AI, automatic Git operations, or execution of commands found in notes.
- **Responsive TUI.** The Bubble Tea Hub adapts from three panes to one below 70 columns and never blocks first render on an index scan.
- **Bring your own editor.** Full editing is delegated to `$EDITOR`; the TUI does not reimplement a text editor.

## Workflows

- `c` and `C` capture into the project or global monthly inbox.
- `n` creates a project note; `d` creates a dated decision record.
- `/` searches the disposable index across registered project stores.
- `e` suspends the Hub and opens the selected Markdown file with the configured editor.
- `now.md` is scaffolded once and always appears first in the project view.

The Hub uses Bubble Tea and adapts from three panes at 110 columns to one active pane below 70 columns. Markdown preview is rendered in process with Glamour; an external `glow` binary is not required.

## Installation

Herdr Logbook currently installs from source. You need Herdr 0.7.0 or newer, Git, and Go 1.25 or newer (see [`go.mod`](go.mod)).

### macOS, Linux, and WSL

```bash
git clone https://github.com/Resetnak/herdr-logbook.git
cd herdr-logbook
mkdir -p bin
go build -o bin/herdr-logbook ./cmd/herdr-logbook
herdr plugin link "$(pwd)" --enabled
herdr plugin action invoke open --plugin herdr-logbook
```

### Windows PowerShell

```powershell
git clone https://github.com/Resetnak/herdr-logbook.git
Set-Location herdr-logbook
New-Item -ItemType Directory -Force bin | Out-Null
go build -o bin\herdr-logbook.exe .\cmd\herdr-logbook
herdr plugin link (Get-Location) --enabled
herdr plugin action invoke open-windows --plugin herdr-logbook
```

`herdr plugin link` does not run build commands. Build the binary first.

After a tagged release is published, Herdr can install it directly:

```bash
herdr plugin install Resetnak/herdr-logbook --ref vX.Y.Z -y
```

Release packaging is configured for Linux, macOS, and Windows on amd64 and arm64. The installers download the exact manifest version and verify `checksums.txt`. A public `v0.1.0` release must not be claimed until the manual compatibility matrix in [docs/herdr-compatibility.md](docs/herdr-compatibility.md) is complete.

## Commands

```text
herdr-logbook tui --view now|project|global|all [--project-root PATH] [--editor CMD]
herdr-logbook capture --scope project|global [--text TEXT | --stdin | --selected]
herdr-logbook decision [--title TEXT] [--project-root PATH]
herdr-logbook init --storage central|repo [--project-root PATH]
herdr-logbook doctor [--json] [--project-root PATH]
herdr-logbook keybinds
herdr-logbook paths [--json] [--project-root PATH]
herdr-logbook index rebuild [--project-root PATH]
herdr-logbook version
```

Without a capture source, `capture` opens the textarea UI. Without `--title`, `decision` prompts for a title. Decision creation opens the configured external editor unless `--no-edit` is used for automation.

Exit codes carry meaning: `2` usage, `3` context/state resolution, `4` storage lock/write, `5` Herdr context, `6` editor.

## Storage and project resolution

Central storage is the default and leaves repositories untouched:

```text
$HERDR_PLUGIN_STATE_DIR/
├── store/projects/p_<sha256>/{now.md,inbox,notes,decisions}
├── store/global/{inbox,notes,decisions}
├── registry/projects.toml
└── cache/index-v1.json
```

`init --storage repo` explicitly opts into `.herdr/logbook/` and prints a suggested ignore rule; it never edits `.gitignore`. Initialization preserves every existing non-empty scaffold file. Cache files are disposable and rebuild from Markdown with `index rebuild`.

Context priority is explicit `--project-root`, Herdr worktree path, focused pane cwd, workspace cwd, `--cwd`, then process cwd. Git worktrees share identity through a credential-free remote fingerprint or Git common directory. Non-Git projects use the canonical path. `.herdr-logbook.toml` may select a monorepo subproject but cannot escape the repository through traversal or symlinks.

## Configuration

Herdr and Herdr Logbook use separate configuration files.

### Choose the editor

Find the plugin configuration directory with `herdr plugin config-dir herdr-logbook`, then create or edit its `config.toml`:

```toml
[editor]
command = ["nvim"]
```

Arguments are separate array items, for example `command = ["code", "--wait"]`. Editor precedence is `tui --editor CMD`, `editor.command`, `$HERDR_LOGBOOK_EDITOR`, `$VISUAL`, `$EDITOR`, then platform defaults. Commands are executed as argv, never interpolated through a shell.

### Add a Herdr shortcut

Add the plugin action to Herdr’s `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+m"
type = "plugin_action"
command = "herdr-logbook.open"
description = "Herdr Logbook"
```

Use `herdr-logbook.open-windows` on Windows. Validate and reload the file with:

```bash
herdr config check
herdr server reload-config
```

`--editor` belongs to the Logbook binary, so it works for direct launches such as `herdr-logbook tui --editor nvim`. Do not append it to `herdr plugin action invoke`; Herdr does not forward action arguments. To override the editor for one Herdr shortcut, pass it to the pane as an environment value:

```toml
[[keys.command]]
key = "prefix+m"
type = "shell"
command = "herdr plugin pane open --plugin herdr-logbook --entrypoint hub --placement split --direction right --focus --env HERDR_LOGBOOK_EDITOR=nvim"
description = "Herdr Logbook"
```

Unknown configuration keys produce warnings. Invalid types and values fail explicitly. User configuration is not rewritten during compatible migrations.

## Privacy and data safety

Normal runtime has no telemetry, network requests, cloud sync, AI calls, automatic Git operations, or execution of commands from notes. Selected terminal text never appears in diagnostics. Git credentials are removed before registry and diagnostic output. Captures use bounded locks, same-directory temporary files, filesystem sync, and atomic replacement.

The index reads only Markdown under known memory roots, skips hidden directories and symlinks, and self-heals after cache corruption. Removing the plugin does not remove Markdown data.

## Limits and roadmap

Herdr Logbook is not a general knowledge-management system, collaboration service, sync engine, browser UI, or custom text editor. Archive/trash, saved searches, templates, backlinks, and optional external Glow integration are deferred until the core workflow has real usage evidence. AI, cloud sync, and graph visualization are explicit non-goals.

Screenshots or a terminal recording will be added only after the real-host release matrix passes.

## Documentation

- [CHANGELOG.md](CHANGELOG.md) — release notes.
- [SECURITY.md](SECURITY.md) — vulnerability reporting and the runtime security model.
- [CONTRIBUTING.md](CONTRIBUTING.md) — development workflow and verification steps.
- [docs/herdr-compatibility.md](docs/herdr-compatibility.md) — the real-host compatibility matrix.

## License

[MIT](LICENSE) © 2026 Alexandr Rešetňak
