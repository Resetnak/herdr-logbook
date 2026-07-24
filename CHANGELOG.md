# Changelog

All notable changes will be documented here. The project has not published a stable release.

## Unreleased

- Added a "How It Compares" section (Obsidian/Logseq, zk/nb, in-repo `TODO.md`) to both READMEs.
- Added `go install` as a standalone-CLI install path and an invitation for Linux/Windows platform-verification reports to both READMEs.
- Release pages now include install instructions and a checksum/platform-status note (GoReleaser header/footer).
- Bug report template now asks for `herdr-logbook doctor --json` output (paths and status only, never note content).
- Re-recorded the demo GIF to show the real Herdr flow: Claude Code in a pane, `prefix+m` splits in the Logbook Hub. `cassette.tape` now drives a throwaway, fully isolated Herdr session and cleans up after itself.

## v0.0.2 — 2026-07-24

Documentation and dependency-hygiene pre-release. No runtime behavior change. Still verified only on macOS arm64 (see [docs/herdr-compatibility.md](docs/herdr-compatibility.md)); not a stable release or a v0.1.0 platform-support claim.

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
