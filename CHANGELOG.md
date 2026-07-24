# Changelog

All notable changes will be documented here. The project has not published a stable release.

## Unreleased

## v0.0.2 — 2026-07-24

Documentation and dependency-hygiene pre-release. No runtime behavior change. Still verified only on macOS arm64 (see [docs/herdr-compatibility.md](docs/herdr-compatibility.md)); not a stable release or a v0.1.0 platform-support claim.

- Reworked README with a header navbar, badges, and a section index; added a Czech translation (`README.cs.md`) with a language switcher.
- Added a Quick Start section and a demo GIF (`assets/demo.gif`, rendered from `cassette.tape`) to both READMEs.
- Corrected the Herdr prerequisite link to point at the real project ([herdr.dev](https://herdr.dev)) and added a Platform status callout scoping verified support to macOS arm64.
- Corrected the documented minimum Go version to match `go.mod`.
- Bumped transitive `golang.org/x` dependencies so `govulncheck` reports zero findings.
- Guarded the CodeQL workflow so CI stays green while the repo is private (runs automatically once public).
- Added a maintainer release checklist (`docs/RELEASE.md`).

## v0.0.1 — 2026-07-22

First pre-release. Verified against Herdr 0.7.5 on macOS arm64 only; other platforms are cross-compiled but untested (see [docs/herdr-compatibility.md](docs/herdr-compatibility.md)). Not a stable release and not a v0.1.0 platform-support claim.

- Added project resolution, central and repository-local stores, safe scaffolding, locks, and atomic writes.
- Added project, global, selected-text, and interactive capture workflows.
- Added the responsive Logbook Hub, Glamour preview, authoring, external editor integration, and cross-project search cache.
- Added diagnostics, release packaging, checksum-verifying installers, and multi-platform CI configuration.
