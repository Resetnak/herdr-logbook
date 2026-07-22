# Changelog

All notable changes will be documented here. The project has not published a stable release.

## Unreleased

## v0.0.2 — 2026-07-22

Documentation and dependency-hygiene pre-release. No runtime behavior change. Still verified only on macOS arm64 (see [docs/herdr-compatibility.md](docs/herdr-compatibility.md)); not a stable release or a v0.1.0 platform-support claim.

- Reworked README with a header navbar, badges, and a section index; added a Czech translation (`README.cs.md`) with a language switcher.
- Corrected the documented minimum Go version to match `go.mod`.
- Bumped transitive `golang.org/x` dependencies so `govulncheck` reports zero findings.

## v0.0.1 — 2026-07-22

First pre-release. Verified against Herdr 0.7.5 on macOS arm64 only; other platforms are cross-compiled but untested (see [docs/herdr-compatibility.md](docs/herdr-compatibility.md)). Not a stable release and not a v0.1.0 platform-support claim.

- Added project resolution, central and repository-local stores, safe scaffolding, locks, and atomic writes.
- Added project, global, selected-text, and interactive capture workflows.
- Added the responsive Logbook Hub, Glamour preview, authoring, external editor integration, and cross-project search cache.
- Added diagnostics, release packaging, checksum-verifying installers, and multi-platform CI configuration.
