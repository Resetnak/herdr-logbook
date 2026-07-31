# Changelog

All notable changes will be documented here. The project has not published a stable release.

## Unreleased

- **Security:** a cloned repository can no longer choose where your notes are
  stored. `storage` in a repository's `.herdr-logbook.toml` is ignored, so a
  crafted repository cannot redirect captures into its own worktree and have
  them pushed to its remote. Repo-local storage stays an explicit opt-in through
  your own config or `--storage`, which are unchanged. A stale `storage` key in
  an existing checkout is ignored rather than rejected.
- **Security:** notes can no longer carry terminal escape sequences to your
  screen. Captures and `now` tasks reject terminal control characters, and note
  titles and previews strip them, so a note body holding OSC 52 can no longer
  rewrite your system clipboard when the Hub paints it. Newlines and tabs are
  unaffected, and CRLF input is normalised rather than refused.
- **Security:** `now.md` and the monthly inbox are no longer read or replaced
  through a symlink, matching the indexer and note loader, which already
  refused them. A repository-supplied `display_name` is stripped of control
  characters and bounded to 120 characters.
- Made saving a note stop re-reading every note in every registered project: the
  rescan now carries over entries whose size and modification time are unchanged
  (125 ms → 5.5 ms and 115 MB → 2.3 MB of allocations over a 1,260-note,
  20-project store). `index rebuild` still forces a full cold read.
- Removed the blocking index-cache read from Hub startup; the same rebuild was
  already scheduled as a command, and the status bar shows `indexing` until it
  lands. The cache is also no longer written with indentation.
- Made search run on a 90 ms debounce instead of on every keystroke, so typing a
  query no longer pays a full scan of every indexed note body per character.
  `Enter` still applies the query immediately.
- Documented the trust boundaries and the detail `doctor --json` prints in
  [SECURITY.md](SECURITY.md).
- Added an activity digest: `s` in the Hub shows a four-week activity heatmap,
  a day streak and a standup summary for today or the past week, with `y`
  copying the standup Markdown to the clipboard. The same report is available
  headless as `herdr-logbook digest [--days N] [--json]`.
- Removed configuration keys that never affected runtime behavior; existing
  pre-release configurations now receive unknown-key warnings for those lines.
  Invalid `ui.theme` and `ui.preview_style` values are rejected instead of
  silently falling back.
- Added one-command Bash and PowerShell build/link scripts for contributors,
  structured real-host compatibility reports, and a pinned Staticcheck version
  in CI.

## v0.0.6 — 2026-07-30

An audit-hardening release: project resolution, capture and authoring, Hub responsiveness, search, and Markdown edge cases are more robust. Pre-release. macOS arm64 remains the only host verified against the checklist in [docs/herdr-compatibility.md](docs/herdr-compatibility.md), with WSL 2 (`linux/amd64`) reported working; not a stable release or a v0.1.0 platform-support claim.

- Fixed project resolution for repositories whose `origin` is a local path, `file://` URL, or another remote without a host; these now fall back to Git-common-directory or path identity instead of making every command fail.
- Fixed `Ctrl+E` so capture and authoring open the exact file written, including an existing monthly inbox and the in-place `now.md`.
- Kept the Hub responsive by caching rendered previews, avoiding an eager terminal-style probe, and moving note reloads, captures, and authoring writes off the Bubble Tea event loop. Preview-panel `j`/`k`/`g`/`G` now scroll without also changing the selected note.
- Reduced search allocations by matching note bodies case-insensitively in place, and kept snippets valid UTF-8 at truncation boundaries.
- Stopped Markdown heading markers from becoming tags, capped generated note and decision slugs at 80 runes, restored headers in empty inbox files, and preserved titles after very long Markdown lines.
- Completed CLI usage and keybind output, including project-filter key `p`, and restored cursor blinking when the standalone capture TUI opens.

## v0.0.5 — 2026-07-29

A platform-status update: WSL 2 is no longer listed as untested. Pre-release. macOS arm64 remains the only host verified against the checklist in [docs/herdr-compatibility.md](docs/herdr-compatibility.md); not a stable release or a v0.1.0 platform-support claim.

- Recorded a WSL 2 (Windows) report: the `linux/amd64` build runs against a real Herdr host. The compatibility matrix now distinguishes `verified` (macOS, checklist run) from `reported working` (WSL 2, hands-on user report with the distro, install path, and exercised actions not recorded). Both READMEs, the badge, the GoReleaser release footer, and the release checklist were reworded to match; the bug-report template lists WSL 2 as an OS example. Native Linux and native Windows (PowerShell) remain cross-compiled and unverified.
- Documented that WSL 2 users should take the Linux archive rather than the Windows one, since Herdr and the plugin run inside the WSL distro.

## v0.0.4 — 2026-07-28

The shortcut you open the Logbook with now closes it too, and Hub search stops serving stale results. Pre-release. Still verified only on macOS arm64 (see [docs/herdr-compatibility.md](docs/herdr-compatibility.md)); not a stable release or a v0.1.0 platform-support claim.

- The `open` and `capture` actions now toggle instead of stacking panes: invoking one while its pane is open but unfocused focuses that pane, and invoking it again from there closes it. Panes are matched by the label Herdr reports (`herdr pane list`); if the listing cannot be read, the action falls back to opening a pane as before.
- Documented how to update the plugin in both READMEs. Herdr has no `plugin update` command, so re-running `herdr plugin install` (or the script installer, or a pull plus rebuild for a linked checkout) is the update path.
- Fixed Hub search staying stale after captures, new notes, editor changes, and manual reloads. Search now updates without restarting Logbook, while overlapping index rebuilds are coalesced to avoid redundant full-store scans.
- Improved empty-state guidance across Hub scopes and failed searches so the UI shows the relevant action or index failure instead of suggesting an unrelated capture or reporting no matches from stale results.
- Fixed `Ctrl+E` in the current-task modal opening a freshly created monthly inbox instead of `now.md`. The Hub picked the editor target by looking for a note path it had not seen before, but `now.md` is rewritten in place and so never appears as a new path.
- Raised test coverage from 73% to 90.9% and lifted the CI coverage floor from 70% to 90%. Both README badges now match. New tests cover the Hub's key handling, note loading and classification, the search index, capture formatting, editor resolution, project resolution and registry, config validation, storage locking and atomic writes, and every documented CLI exit code.
- Extracted the Hub's note-authoring and editor-launch logic out of the Bubble Tea closures (`authorFromHub`, `editorCommandFor`) so both are testable without a terminal. No behaviour change.

## v0.0.3 — 2026-07-26

The active task becomes a first-class command, and switching tasks writes your work journal for you. Pre-release. Still verified only on macOS arm64 (see [docs/herdr-compatibility.md](docs/herdr-compatibility.md)); not a stable release or a v0.1.0 platform-support claim.

- Added the active-task workflow: `herdr-logbook now` prints the current task, `herdr-logbook now "TASK"` sets it, and `t` in the Hub opens the same modal. Switching tasks archives the one being replaced into the monthly inbox as `Task done: …`, which turns `now.md` into a work journal without any extra bookkeeping. Only the `## Current task` section of `now.md` is rewritten; every other section is preserved.
- Documented the active task in both READMEs (feature list, keybind tables, and a dedicated section) and added it to the `?` help screen and `herdr-logbook keybinds`.
- Demo GIF now ends on the active-task flow: `t`, type a task, `Ctrl+S`, with the displaced task visible in the Project Inbox as `Task done: …`.
- Fixed the standalone script installers (`curl … | bash`, `irm … | iex`): without a plugin checkout they now read the released version from the manifest on `main` and install the binary to `~/.local/bin` instead of failing on a missing `herdr-plugin.toml`.

## v0.0.2 — 2026-07-24

Documentation, dependency-hygiene, and a capture-modal rendering fix. Pre-release. Still verified only on macOS arm64 (see [docs/herdr-compatibility.md](docs/herdr-compatibility.md)); not a stable release or a v0.1.0 platform-support claim.

- Added a "How It Compares" section (Obsidian/Logseq, zk/nb, in-repo `TODO.md`) to both READMEs.
- Added `go install` as a standalone-CLI install path and an invitation for Linux/Windows platform-verification reports to both READMEs.
- Release pages now include install instructions and a checksum/platform-status note (GoReleaser header/footer).
- Bug report template now asks for `herdr-logbook doctor --json` output (paths and status only, never note content).
- Re-recorded the demo GIF to show the real Herdr flow: Claude Code in a pane, `prefix+m` splits in the Logbook Hub. `cassette.tape` now drives a throwaway, fully isolated Herdr session and cleans up after itself.
- Fixed the capture modal rendering: the box was two columns narrower than the textarea rows, so lipgloss re-wrapped them — text containing `-` visually jumped to the next line and the cursor-line highlight left artifacts. Data on disk was never affected.
- Reworked README with a header navbar, badges, and a section index; added a Czech translation (`README.cs.md`) with a language switcher.
- Added a Quick Start section and a demo GIF (`assets/demo.gif`, rendered from `cassette.tape`) to both READMEs.
- Corrected the Herdr prerequisite link to point at the real project ([herdr.dev](https://herdr.dev)) and added a Platform status callout scoping verified support to macOS arm64.
- Corrected the documented minimum Go version to match `go.mod`.
- Bumped transitive `golang.org/x` dependencies so `govulncheck` reports zero findings.
- Guarded the CodeQL workflow so CI stays green while the repo is private (runs automatically once public).
- Added a `CODE_OF_CONDUCT.md` (Contributor Covenant 2.1) and a `.github/FUNDING.yml` sponsor button.
- Added a test-coverage badge to both READMEs backed by a CI coverage floor that fails below 70%.
- Reworded the installer callout so it no longer implies verified support on Linux/Windows; only macOS (arm64) is verified against a real Herdr host.

## v0.0.1 — 2026-07-22

First pre-release. Verified against Herdr 0.7.5 on macOS arm64 only; other platforms are cross-compiled but untested (see [docs/herdr-compatibility.md](docs/herdr-compatibility.md)). Not a stable release and not a v0.1.0 platform-support claim.

- Added project resolution, central and repository-local stores, safe scaffolding, locks, and atomic writes.
- Added project, global, selected-text, and interactive capture workflows.
- Added the responsive Logbook Hub, Glamour preview, authoring, external editor integration, and cross-project search cache.
- Added diagnostics, release packaging, checksum-verifying installers, and multi-platform CI configuration.
