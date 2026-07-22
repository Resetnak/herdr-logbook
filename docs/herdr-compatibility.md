# Herdr Compatibility

This document records observed Herdr behavior for the Phase 0 compatibility spike. It is evidence, not a cross-platform support claim.

## Verified locally

- Date: 2026-07-22
- Herdr: 0.7.5
- Host: macOS Apple Silicon (`darwin/arm64`)
- Plugin API: manifest accepted with `min_herdr_version = "0.7.0"`
- Plugin root with spaces and Unicode: verified using `/private/tmp/herdr-logbook žluťoučký.<suffix>`

The accepted popup command is:

```text
herdr plugin pane open \
  --plugin herdr-logbook \
  --entrypoint compatibility \
  --placement overlay \
  --focus \
  --cwd <origin-cwd>
```

An action runs without a TTY. It receives `HERDR_BIN_PATH`, `HERDR_PLUGIN_ROOT`, and `HERDR_PLUGIN_CONTEXT_JSON`. The launcher resolves the originating directory in Go and opens the declared pane, which receives a TTY.

The pane received these paths from Herdr:

```text
HERDR_BIN_PATH
HERDR_PLUGIN_ROOT
HERDR_PLUGIN_CONFIG_DIR
HERDR_PLUGIN_STATE_DIR
```

It also received workspace, tab, and pane IDs. The invocation context contained workspace and focused-pane cwd values; both resolved to the originating repository. `invocation_source` was `cli` for the action and `api` inside the opened pane. The diagnostic only reports context keys and a selected-text presence flag, never selected content or clicked URLs.

Herdr 0.7.5 exposes no width or height option for `plugin pane open`; the requested `92%` by `86%` popup size cannot currently be set through the documented CLI. The spike uses Herdr's default overlay size.

## Compatibility matrix

| Platform | Build | Real Herdr action/pane | Notes |
|---|---:|---:|---|
| macOS arm64 | verified | verified | Herdr 0.7.5, Bash launcher |
| macOS amd64 | cross-compile only | not tested | Intel host required |
| Linux amd64 | cross-compile only | not tested | Linux Herdr host required |
| Linux arm64 | cross-compile only | not tested | Linux arm64 Herdr host required |
| Windows amd64 | cross-compile only | not tested | PowerShell 5.1 and 7 required |
| Windows arm64 | cross-compile only | not tested | Windows arm64 Herdr host required |
| WSL 2 | Linux build only | not tested | WSL Herdr host required |

The Windows manifest entry uses PowerShell as the pane executable and invokes the Go binary by absolute `HERDR_PLUGIN_ROOT` path. Herdr accepted the manifest on macOS, but this is not proof that CreateProcess, PowerShell 5.1, verbatim paths, and pane context propagation work on Windows.

## Reproduction

```bash
go build -o bin/herdr-logbook ./cmd/herdr-logbook
herdr plugin link "$(pwd)" --enabled
herdr plugin action invoke open --plugin herdr-logbook
herdr plugin log list
herdr pane list
```

`herdr plugin link` does not run build commands. Build the binary first for local development.

## Gate result

The macOS arm64 launcher contract is suitable for subsequent local core-domain work. Production launch architecture and cross-platform support remain blocked on real Linux, Windows, WSL, Intel macOS, PowerShell 5.1, and PowerShell 7 runs.
