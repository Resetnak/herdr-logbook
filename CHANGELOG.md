# Changelog

All notable changes will be documented here. The project has not published a stable release.

## Unreleased

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
