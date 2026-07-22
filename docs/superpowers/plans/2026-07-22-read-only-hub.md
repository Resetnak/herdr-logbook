# Herdr Logbook Read-only Hub Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a keyboard-only responsive Bubble Tea Hub that lists project/global Markdown and previews the selected note with Glamour.

**Architecture:** A pure note loader returns ordered metadata with `now.md` first. `internal/app` owns one Bubble Tea model whose view switches between three-, two-, and one-pane layouts by width; Markdown rendering is recomputed for preview width. Capture textarea reuses Bubbles and commits through the existing capture package.

**Tech Stack:** Bubble Tea, Bubbles, Lip Gloss, Glamour, Go standard library.

### Task 1: Notes loader

- [ ] Test `.md` filtering, hidden/symlink exclusion, title fallback, scope classification, and `now.md` first.
- [ ] Implement `internal/markdown` title extraction and `internal/app` note loading.
- [ ] Run package tests.

### Task 2: Responsive Hub model

- [ ] Test wide, medium, narrow, empty, navigation, help, and preview-width updates through `Update` and `View`.
- [ ] Implement scopes, list, preview, Tab/Shift+Tab, arrows/j/k/h/l, g/G, Enter, `?`, `r`, `v`, and `q`.
- [ ] Render Markdown through Glamour with no external `glow` dependency.

### Task 3: CLI and capture textarea

- [ ] Add failing CLI tests for `tui --view` and capture without text entering textarea mode.
- [ ] Wire project/global stores into the Hub and reuse `capture.Append` on textarea submit.
- [ ] Run tests, race, vet, staticcheck, six cross-builds, and live Herdr TTY smoke.
