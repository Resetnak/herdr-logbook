# Herdr Logbook Capture Workflows Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement project, global, stdin, and selected-terminal capture into concurrency-safe monthly Markdown inboxes.

**Architecture:** `internal/capture` owns pure Markdown formatting and locked append. The CLI resolves context through the existing Phase 1 core, enforces configured size limits, and passes content and metadata without logging it.

**Tech Stack:** Go standard library plus existing `storage.WithLock` and `storage.AtomicWrite`.

## Global Constraints

- Monthly files are `inbox/YYYY-MM.md` and Markdown remains canonical.
- Selected content is limited to `524288` bytes by default and never appears in diagnostics or logs.
- Every append re-reads under the target lock and atomically replaces the file.
- Empty metadata rows are omitted and multiline selections use a collision-safe fence.

### Task 1: Capture formatting

**Files:**
- Create: `internal/capture/format_test.go`
- Create: `internal/capture/format.go`

- [x] Write failing tests for plain capture metadata, omitted empty metadata, one-line selection, multiline selection, and embedded backticks.
- [x] Implement `capture.Format(Entry) (string, error)` and `capture.Fence(string) string`.
- [x] Run `go test ./internal/capture`.

### Task 2: Locked monthly append

**Files:**
- Create: `internal/capture/capture_test.go`
- Create: `internal/capture/capture.go`

- [x] Write failing tests for monthly filenames, initial newline handling, size rejection, and concurrent capture preservation.
- [x] Implement `capture.Append(Request) (string, error)` with bounded flock and atomic replacement.
- [x] Run `go test -race ./internal/capture`.

### Task 3: Capture CLI

**Files:**
- Modify: `internal/herdr/context.go`
- Modify: `internal/herdr/context_test.go`
- Modify: `cmd/herdr-logbook/main.go`
- Modify: `cmd/herdr-logbook/main_test.go`
- Modify: `README.md`

- [x] Write failing tests for `--text`, `--stdin`, `--selected`, project/global scope, byte limits, and argument conflicts.
- [x] Implement `herdr.SelectedText`, CLI flag parsing, project/global target selection, and stable exit codes.
- [x] Run full tests, race detector, vet, staticcheck, cross-builds, and a temp-state CLI smoke test.
