# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Read `AGENTS.md` first — it holds the authoritative structure, working rules, and verification steps. `HERDR_LOGBOOK_AGENT_BRIEF.md` is the product source of truth (user-owned; edit only when a task asks for it). This file adds the big-picture architecture and the non-obvious bits.

## What this is

Herdr Logbook — a local, offline, Markdown-first working-memory plugin for the Herdr terminal tool. A Bubble Tea TUI ("Hub") plus subcommands that capture notes, decisions, and an active-task `now.md`, storing everything as plain Markdown. No telemetry, cloud sync, AI, SQLite, auto-git, or execution of commands found in notes — these are hard product boundaries, not TODOs.

Naming trap: the Go module is `github.com/Resetnak/herdr-logbook` (the repo dir), but the product and binary are `herdr-logbook`.

## Commands

There is no Makefile — use raw Go tooling.

```bash
go test ./...                                        # tests
go test -race ./...                                  # race detector (CI runs this)
go vet ./... && staticcheck ./... && govulncheck ./...  # CI lint/security gates
go build -o bin/herdr-logbook ./cmd/herdr-logbook     # build
go test ./internal/capture/ -run TestAppend         # single package / test
```

CI (`.github/workflows/ci.yml`) runs the full gate on linux/macOS/windows and shell-parses the install/open scripts. Match it before handoff.

## Running the binary

It expects Herdr's environment, so direct runs need env vars — `HERDR_PLUGIN_STATE_DIR` is **required** (`loadCore` errors without it), `HERDR_PLUGIN_CONFIG_DIR` and `HERDR_PLUGIN_CONTEXT_JSON` are optional. In practice, link it into Herdr instead:

```bash
herdr plugin link "$(pwd)" --enabled   # does NOT build — build the binary first
herdr plugin action invoke open --plugin herdr-logbook
```

Subcommands: `tui`, `capture`, `decision`, `init`, `doctor [--json]`, `paths`, `index rebuild`, `keybinds`, `version`, `compatibility`. See README for flags. Exit codes carry meaning: 2 usage, 3 context/state resolution, 4 storage lock/write, 5 Herdr context, 6 editor.

## Architecture

CLI in `cmd/herdr-logbook/main.go` is a flat command switch; each `runX` builds `coreState` via `loadCore` (context → config → project identity → storage layout), then calls into `internal/`:

- `internal/app/` — Bubble Tea Hub, capture modal, search, authoring, external-editor flow. Full Markdown editing is delegated to the user's `$EDITOR`; do not build a second editor into the TUI. Keep it responsive below 70 columns and never block first render on a full index scan.
- `internal/project/` — project identity + resolution. Context priority: explicit `--project-root` → Herdr worktree → focused pane cwd → workspace cwd → `--cwd` → process cwd. Git worktrees share identity via a credential-free remote fingerprint.
- `internal/storage/` — safe paths, bounded locks (`gofrs/flock`), same-directory atomic writes. Every user-data replacement goes through a lock + atomic write; central storage is the default and repo-local (`.herdr/logbook/`) is explicit opt-in only.
- `internal/capture/` — monthly-inbox append with size limits.
- `internal/index/` — disposable JSON search cache; Markdown stays canonical and the cache self-heals / rebuilds via `index rebuild`.
- `internal/author/`, `internal/editor/`, `internal/markdown/`, `internal/herdr/` — note creation, argv-only editor resolution (never shelled out), metadata parsing, and Herdr context/env reading.

## Gotchas

- Editor commands are executed as argv, never through a shell — keep it that way.
- Never log selected terminal text or unsanitized Git remotes (privacy guarantee; credentials are stripped before registry/diagnostic output).
- Preview renders in-process with Glamour; `glow` is optional, not required.
- Do not claim `v0.1.0` or platform support until the real-host matrix in `docs/herdr-compatibility.md` passes. Don't tag, publish, or run destructive Herdr ops without explicit authorization.
- Treat `.tokensave/` as user-owned input.
