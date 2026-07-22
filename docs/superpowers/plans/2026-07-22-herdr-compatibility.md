# Herdr Compatibility Spike Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use `superpowers:executing-plans` to implement this plan task-by-task.

**Goal:** Prove the Herdr 0.7 action-to-overlay contract with a safe diagnostic Go binary.

**Architecture:** The Go package `internal/herdr` owns context parsing and redaction. `cmd/herdr-logbook` only maps command-line input to that package. The manifest and one thin launcher per platform invoke the binary through Herdr's supported plugin CLI.

**Tech Stack:** Go 1.24-compatible standard library, Herdr plugin API v1, Bash, Windows PowerShell.

## Global Constraints

- Plugin ID is `herdr-logbook` and minimum Herdr version is `0.7.0`.
- Runtime makes no network requests and emits no selected terminal content.
- Do not proceed to production UI until the compatibility matrix is verified.

### Task 1: Context diagnostic

**Files:**
- Create: `go.mod`
- Create: `internal/herdr/context_test.go`
- Create: `internal/herdr/context.go`
- Create: `cmd/herdr-logbook/main.go`

**Interfaces:**
- Produces: `herdr.ReadContext(func(string) string) (Context, error)` and `herdr.ResolveWorkingDirectory(Context, string) string`.

- [x] Write tests proving environment parsing, redaction, invalid JSON handling, and cwd precedence.
- [x] Run `go test ./internal/herdr` and verify it fails because the implementation is absent.
- [x] Add the minimum standard-library implementation.
- [x] Run `go test ./internal/herdr` and verify it passes.

### Task 2: Manifest and launchers

**Files:**
- Create: `herdr-plugin.toml`
- Create: `scripts/open.sh`
- Create: `scripts/open.ps1`
- Create: `docs/herdr-compatibility.md`

**Interfaces:**
- Consumes: `herdr-logbook compatibility` from Task 1.
- Produces: Herdr actions `herdr-logbook.open` and `herdr-logbook.open-windows`.

- [x] Add the minimal v1 manifest and platform launchers using absolute plugin-root paths.
- [x] Run shell syntax validation and cross-compile the diagnostic binary for all requested targets.
- [x] Link the plugin locally and verify the registered macOS action and overlay behavior.
- [x] Record exact observed flags, context, platform results, and unresolved matrix rows.

### Task 3: Final verification

**Files:**
- Modify: `README.md`

- [x] Document local build, link, and compatibility-spike usage.
- [x] Run `go test ./...`, `go vet ./...`, manifest parsing, shell syntax validation, and repository diff review.
- [x] Compare the result against Phase 0 deliverables in `HERDR_LOGBOOK_AGENT_BRIEF.md` and report every unverified platform without claiming support.
