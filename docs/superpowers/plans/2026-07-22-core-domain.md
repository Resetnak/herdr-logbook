# Herdr Memory Core Domain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Phase 1 configuration, project identity, storage, registry, and diagnostic CLI without writing private data into repositories by default.

**Architecture:** Pure packages own configuration, project resolution, storage, registry, and editor discovery. The CLI composes those packages and emits either human-readable output or stable JSON. Markdown remains canonical; registry and cache files are replaceable metadata.

**Tech Stack:** Go 1.24 standard library, `github.com/pelletier/go-toml/v2 v2.4.3`, `github.com/gofrs/flock v0.13.0`.

## Global Constraints

- Plugin API v1 and `min_herdr_version = "0.7.0"`.
- Central storage is the default; repository-local storage requires explicit `init --storage repo`.
- Runtime performs no network requests, logs no selected text, and persists no Git credentials.
- Writes use same-directory temporary files, flush, close, and atomic replacement under bounded locks.
- Project identity priority is sanitized Git remote, Git common directory, then canonical project path.

---

### Task 1: Versioned configuration

**Files:**
- Modify: `go.mod`
- Create: `internal/config/config_test.go`
- Create: `internal/config/config.go`

**Interfaces:**
- Produces: `config.Default() Config`, `config.Load(path string) (Config, []string, error)`, and `Config.Validate() error`.

- [x] Write tests for missing-file defaults, valid overrides, invalid types/values, version migration from zero to one in memory, and unknown-key warnings.
- [x] Run `go test ./internal/config` and verify missing symbols fail the build.
- [x] Add the two pinned dependencies and implement the smallest typed TOML loader; treat `toml.StrictMissingError` as warnings and every other decode error as fatal.
- [x] Run `go test ./internal/config` and verify all tests pass.

### Task 2: Project resolution and stable identity

**Files:**
- Create: `internal/project/resolve_test.go`
- Create: `internal/project/resolve.go`
- Create: `internal/project/identity_test.go`
- Create: `internal/project/identity.go`

**Interfaces:**
- Produces: `project.Resolve(ResolveOptions) (Project, error)`, `project.SanitizeRemote(string) (string, error)`, and `project.StableID(string) string`.
- `ResolveOptions` carries explicit root, worktree, focused pane cwd, workspace cwd, fallback cwd, and a command runner.
- `Project` carries ID, display name, canonical root, branch, sanitized fingerprint, and observed roots.

- [x] Write tests for context priority, Git/non-Git roots, HTTPS/SCP remotes with credentials, no-remote identity, spaces, Unicode, worktree common identity, override containment, and deterministic SHA-256 IDs.
- [x] Run `go test ./internal/project` and verify missing symbols fail the build.
- [x] Implement path selection, `git -C` root/common-dir/remote/branch queries, upward `.git` fallback, remote normalization, override parsing, and boundary validation.
- [x] Run `go test ./internal/project` and verify all tests pass.

### Task 3: Safe storage and locking

**Files:**
- Create: `internal/storage/storage_test.go`
- Create: `internal/storage/storage.go`
- Create: `internal/storage/atomic_test.go`
- Create: `internal/storage/atomic.go`
- Create: `internal/storage/lock_test.go`
- Create: `internal/storage/lock.go`

**Interfaces:**
- Produces: `storage.Resolve(stateDir, projectRoot, projectID string, cfg config.Config, mode string) (Layout, error)`, `storage.Initialize(Layout) error`, `storage.AtomicWrite(path string, data []byte, perm fs.FileMode) error`, and `storage.WithLock(path string, timeout time.Duration, fn func() error) error`.

- [x] Write tests proving central default paths, explicit repo paths, traversal rejection, non-empty `now.md` preservation, atomic replacement, cleanup after write failure, and lock timeout/concurrent exclusion.
- [x] Run `go test ./internal/storage` and verify missing symbols fail the build.
- [x] Implement same-directory atomic replacement, flock-based bounded locking, and idempotent scaffold creation.
- [x] Run `go test -race ./internal/storage` and verify all tests pass.

### Task 4: Project registry

**Files:**
- Create: `internal/project/registry_test.go`
- Create: `internal/project/registry.go`

**Interfaces:**
- Produces: `project.LoadRegistry(path string) (Registry, error)` and `(*Registry).Upsert(Project, storageMode, storePath string, now time.Time)`.

- [x] Write tests for first registration, moved roots, worktree aliases, credential-free TOML, stable timestamps, corrupt files, locking, and atomic save.
- [x] Run the registry tests and verify missing symbols fail the build.
- [x] Implement sorted TOML records, alias merging, atomic save under `registry.lock`, and actionable corruption errors.
- [x] Run `go test -race ./internal/project` and verify all tests pass.

### Task 5: Editor discovery and Phase 1 CLI

**Files:**
- Create: `internal/editor/resolve_test.go`
- Create: `internal/editor/resolve.go`
- Modify: `cmd/herdr-memory/main_test.go`
- Modify: `cmd/herdr-memory/main.go`
- Modify: `README.md`
- Modify: `docs/herdr-compatibility.md`

**Interfaces:**
- Produces CLI commands `init`, `paths`, and `doctor [--json]` while preserving `compatibility`, `resolve-cwd`, and `version`.

- [x] Write tests for editor precedence and CLI JSON that contains project/store diagnostics but no selected text or remote credentials.
- [x] Run targeted tests and verify the new commands fail as unavailable.
- [x] Implement manual standard-library flag parsing, project/config/storage composition, idempotent initialization, registry update, path output, and doctor checks.
- [x] Update README with exact implemented commands and central versus repository-local behavior.
- [x] Run `go test ./...`, `go test -race ./...`, `go vet ./...`, six cross-builds, manifest parsing, shell syntax validation, and live macOS Herdr compatibility action.
