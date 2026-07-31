# Repository Guidelines

## Scope

This repository contains Herdr Logbook (`herdr-logbook`), a local, offline, Markdown-first working-memory plugin for Herdr. Keep changes inside this product boundary. Do not add telemetry, cloud sync, AI APIs, automatic Git operations, SQLite, or execution of commands found in notes.

`README.md`, `config.example.toml`, and the behavior covered by tests are the current product contract. Treat `.tokensave/` as user-owned input unless a task explicitly asks to edit it.

## Structure

- `cmd/herdr-logbook/`: CLI entrypoint and command wiring.
- `internal/app/`: Bubble Tea Hub, capture modal, search, authoring, and editor flow.
- `internal/project/`, `internal/storage/`, `internal/capture/`: project identity, safe paths, locks, and atomic Markdown writes.
- `internal/index/`: disposable JSON search cache; Markdown remains canonical.
- `internal/digest/`: read-only activity roll-up over the stored Markdown (standup report and heatmap).
- `internal/config/`, `internal/author/`, `internal/editor/`, `internal/markdown/`: validated configuration, note creation, safe argv editor resolution, and metadata parsing.
- `scripts/` and `herdr-plugin.toml`: platform launchers and checksum-verifying installers.
- `.github/workflows/`: CI and release packaging.

## Working Rules

Use `rtk` before every shell command. Prefer `rg`/`rtk rg` for search and `apply_patch` for edits. Preserve unrelated working-tree changes. Never log selected terminal content or unsanitized Git remotes. Repository-local storage must remain explicit opt-in. All user-data replacements require bounded locks and same-directory atomic writes.

Full Markdown editing belongs to the configured external editor; do not build another editor into the TUI. Keep Bubble Tea behavior responsive below 70 columns and avoid blocking first render on a full index scan.

## Verification

Run the smallest relevant test first, then before handoff:

```bash
rtk test go test ./...
rtk test go test -race ./...
rtk proxy go vet ./...
rtk proxy staticcheck ./...
rtk proxy bash -n scripts/open.sh scripts/install.sh
```

For release-sensitive changes, cross-build `linux`, `darwin`, and `windows` for `amd64` and `arm64`. Do not claim v0.1.0 or platform support until the real-host matrix in `docs/herdr-compatibility.md` passes. Do not publish, tag, push, or invoke destructive Herdr operations without explicit user authorization.
