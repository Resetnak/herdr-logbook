# Herdr Compatibility Spike Design

## Scope

Phase 0 proves the Herdr 0.7 plugin boundary before product features are built. The repository remains `herdr-logbook`; the product name and plugin ID remain `Herdr Memory` and `herdr-memory` as specified in `HERDR_MEMORY_AGENT_BRIEF.md`.

## Design

A small Go binary reads Herdr-provided environment variables, parses the invocation context, removes selected terminal content, and prints a deterministic JSON diagnostic. A minimal manifest exposes an `open` action and an overlay pane. Thin Bash and PowerShell launchers resolve the Herdr binary and plugin root without interpolating user data into a shell command.

The spike records observed behavior instead of promoting assumptions. macOS Apple Silicon is verified locally against Herdr 0.7.5. Linux, Windows, WSL, Intel macOS, PowerShell 5.1, and PowerShell 7 remain explicitly unverified until they run on those systems.

## Safety

- Diagnostic output never includes selected terminal text.
- The binary performs no network requests and writes no project files.
- Paths and JSON are passed as arguments or environment values, not evaluated as shell source.
- Unsupported or malformed context produces a useful error and exit code 5.

## Verification

Unit tests cover environment parsing, context redaction, invalid JSON, and working-directory precedence. Local integration verifies the manifest can be linked, the action is registered, and the overlay launcher receives the originating working directory.
