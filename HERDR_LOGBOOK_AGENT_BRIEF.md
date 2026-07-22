# Herdr Logbook — Complete Implementation Brief for AI Agent

## Mission

Build a production-ready Herdr plugin called **Herdr Logbook**.

Herdr Logbook is a local, Markdown-first **working memory for developers inside Herdr**. It should automatically understand the active project, make it extremely cheap to capture context, help users resume work, and provide one place to search notes across all projects.

The product is **not** a general-purpose knowledge-management system, an Obsidian clone, an AI second brain, or a custom text editor.

Core promise:

> Capture what matters while working, then find it again in the project where it belongs.

Repository name:

```text
herdr-logbook
```

Initial plugin ID:

```text
herdr-logbook
```

Target:

```text
Herdr plugin API v1
min_herdr_version = "0.7.0"
```

Platforms:

- Linux
- macOS Intel
- macOS Apple Silicon
- Windows
- WSL as Linux

Primary implementation language:

```text
Go
```

---

# 1. Product requirements

## 1.1 The problem

Developers already have Markdown files, Obsidian, Notion, README files, issue trackers, and text editors.

What they often do not have is a small, context-aware memory layer tied directly to the project currently open in their terminal workspace.

The plugin should solve these concrete problems:

- “What was I working on in this project?”
- “What should I do next?”
- “Why did we make this technical decision?”
- “What command, endpoint, port, caveat, or debugging result did I discover?”
- “Where did I write that note from another project?”
- “How can I save selected terminal output without switching applications?”

## 1.2 Main user workflows

The v0.1 release must support:

1. Open the active project’s memory.
2. Show a special `now.md` file first.
3. Capture a short note into the current project.
4. Capture a short note into a global inbox.
5. Capture selected terminal text into the current project.
6. Create a structured technical decision record.
7. Browse project notes.
8. Browse global notes.
9. Search notes across every registered project.
10. Preview Markdown with Glow-quality rendering.
11. Open a note in the user’s real external editor.
12. Diagnose project resolution, paths, configuration, and dependencies.

## 1.3 Product principles

Follow these strictly:

1. Capture must be cheaper than opening another application.
2. Markdown files are the source of truth.
3. Search indexes and caches are disposable.
4. User data must survive upgrades and cache corruption.
5. Normal users must not need Go or another runtime.
6. Project detection must be deterministic and inspectable.
7. The plugin must not silently write private files into repositories.
8. Windows must be treated as a first-class platform from the beginning.
9. The interface must work entirely from the keyboard.
10. Every destructive action must be explicit.
11. Full editing belongs to the user’s existing editor.
12. The plugin must make no network requests during normal runtime.
13. Do not add telemetry.

---

# 2. Explicit non-goals for v0.1

Do not implement:

- AI summaries;
- embeddings;
- semantic vector search;
- cloud sync;
- accounts;
- collaboration;
- browser or mobile UI;
- backlinks graph;
- knowledge graph;
- WYSIWYG Markdown;
- a custom Vim implementation;
- project management boards;
- automatic Git commits;
- automatic `.gitignore` modification;
- command execution from Markdown;
- background daemon;
- home-directory-wide scanning;
- SQLite unless benchmarks prove the file index inadequate;
- branch-specific storage by default;
- custom synchronization protocol.

Do not broaden scope without a documented product decision.

---

# 3. Technology choices

## 3.1 Language

Use Go.

Reasons:

- one self-contained binary;
- good cross-compilation;
- strong filesystem and process APIs;
- good terminal UI libraries;
- simpler long-term maintenance than implementing a text editor in Rust;
- no runtime dependency for users.

## 3.2 TUI libraries

Use:

```text
github.com/charmbracelet/bubbletea
github.com/charmbracelet/bubbles
github.com/charmbracelet/lipgloss
github.com/charmbracelet/glamour
github.com/sahilm/fuzzy
github.com/pelletier/go-toml/v2
```

Use a small cross-platform locking package such as:

```text
github.com/gofrs/flock
```

Use `fsnotify` only if live external-change refresh is implemented and tested reliably.

## 3.3 Glow integration

Do not require the external `glow` binary for previews.

Use **Glamour**, the Markdown renderer behind Glow, directly inside the TUI. This gives Glow-style terminal output without spawning an external process on each render.

External Glow support is optional:

- detect `glow` on `PATH`;
- expose an “Open in Glow” command;
- do not treat Glow as a required dependency.

## 3.4 Editing

Do not build a general text editor.

Use:

1. a built-in multiline textarea for quick capture and small forms;
2. the user’s actual external editor for full Markdown editing.

Editor resolution order:

1. `editor.command` from plugin config;
2. `HERDR_LOGBOOK_EDITOR`;
3. `VISUAL`;
4. `EDITOR`;
5. platform defaults.

Unix fallback candidates:

```text
nvim
vim
vi
nano
```

Windows fallback candidates:

```text
nvim.exe
vim.exe
notepad.exe
```

Configuration should store an argv array:

```toml
[editor]
command = ["nvim"]
```

Do not invoke configured editors through an unsafe shell string.

---

# 4. Required repository structure

Use a structure close to:

```text
herdr-logbook/
├── .github/
│   └── workflows/
│       ├── ci.yml
│       ├── release.yml
│       └── smoke.yml
├── cmd/
│   └── herdr-logbook/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── model.go
│   │   ├── update.go
│   │   ├── view.go
│   │   ├── keymap.go
│   │   ├── commands.go
│   │   └── responsive.go
│   ├── capture/
│   │   ├── capture.go
│   │   ├── format.go
│   │   └── selection.go
│   ├── config/
│   │   ├── config.go
│   │   ├── defaults.go
│   │   └── migrate.go
│   ├── editor/
│   │   ├── resolve.go
│   │   └── launch.go
│   ├── herdr/
│   │   ├── context.go
│   │   ├── client.go
│   │   ├── launcher.go
│   │   └── panes.go
│   ├── index/
│   │   ├── index.go
│   │   ├── cache.go
│   │   ├── search.go
│   │   └── ranking.go
│   ├── markdown/
│   │   ├── render.go
│   │   ├── title.go
│   │   └── tags.go
│   ├── project/
│   │   ├── resolve.go
│   │   ├── identity.go
│   │   ├── git.go
│   │   └── registry.go
│   ├── storage/
│   │   ├── store.go
│   │   ├── paths.go
│   │   ├── atomic.go
│   │   ├── lock.go
│   │   └── scaffold.go
│   ├── testutil/
│   └── version/
├── scripts/
│   ├── install.sh
│   ├── install.ps1
│   ├── open.sh
│   ├── open.ps1
│   ├── capture.sh
│   └── capture.ps1
├── testdata/
├── bin/
│   └── .gitkeep
├── docs/
│   ├── herdr-compatibility.md
│   └── decisions/
├── herdr-plugin.toml
├── config.example.toml
├── go.mod
├── go.sum
├── README.md
├── CONTRIBUTING.md
├── CHANGELOG.md
├── SECURITY.md
└── LICENSE
```

Keep shell and PowerShell scripts thin. Business logic belongs in Go and must be testable without Herdr running.

---

# 5. Storage model

## 5.1 Default mode: central storage

By default, store notes under:

```text
HERDR_PLUGIN_STATE_DIR
```

Do not create files inside the active repository until the user explicitly opts in.

Suggested layout:

```text
$HERDR_PLUGIN_STATE_DIR/
├── store/
│   ├── global/
│   │   ├── inbox/
│   │   │   └── 2026-07.md
│   │   ├── notes/
│   │   └── decisions/
│   └── projects/
│       └── p_<stable-project-id>/
│           ├── project.toml
│           ├── now.md
│           ├── inbox/
│           │   └── 2026-07.md
│           ├── notes/
│           └── decisions/
├── registry/
│   └── projects.toml
├── cache/
│   └── index-v1.json
└── locks/
    ├── registry.lock
    └── p_<stable-project-id>.lock
```

## 5.2 Optional repository-local storage

Allow explicit initialization:

```bash
herdr-logbook init --storage repo
```

Repository-local layout:

```text
<project-root>/.herdr/logbook/
├── now.md
├── inbox/
│   └── 2026-07.md
├── notes/
└── decisions/
```

Rules:

- warn that files may be committed;
- do not modify `.gitignore` automatically;
- offer a command that prints the recommended ignore rule;
- retain a central registry entry so global search can discover the store later.

Recommended ignore entry:

```text
.herdr/logbook/
```

## 5.3 Monthly inbox files

Use:

```text
inbox/YYYY-MM.md
```

Do not use one permanent, endlessly growing inbox file.

Benefits:

- bounded file size;
- faster rendering;
- easier archiving;
- fewer merge conflicts;
- clearer chronological organization.

## 5.4 `now.md`

Each project gets a special resumable context file:

```markdown
# Now

## Current task

Describe the task currently in progress.

## Next steps

- [ ] Add the next concrete action.

## Blockers

- None.

## Context

Record the details that will help you resume later.
```

Never overwrite a non-empty `now.md`.

## 5.5 Decision records

Create:

```text
decisions/YYYY-MM-DD-short-title.md
```

Template:

```markdown
# Decision: Short title

- Date: 2026-07-22
- Status: accepted
- Project: example-project
- Branch: feature/example

## Context

## Decision

## Consequences

### Positive

### Negative

## Follow-up
```

Open the file in the configured external editor after creation.

## 5.6 Markdown metadata

Do not require YAML front matter in v0.1.

Derive:

- title from first H1;
- fallback title from filename;
- note type from directory;
- project from store;
- timestamps from filename or filesystem metadata;
- tags from `#tag` tokens.

Ignore tags inside fenced code blocks where practical.

Markdown files must remain readable in any editor.

---

# 6. Project resolution

Project resolution is core business logic and requires extensive tests.

## 6.1 Context priority

Resolve the active working directory using this priority:

1. CLI `--project-root`;
2. worktree path from Herdr context;
3. `focused_pane_cwd`;
4. `workspace_cwd`;
5. CLI `--cwd`;
6. current process directory.

## 6.2 Root detection

From the selected directory:

1. normalize the path;
2. try:

```bash
git -C <cwd> rev-parse --show-toplevel
```

3. if unavailable or unsuccessful, walk upward for `.git`;
4. otherwise treat the selected directory as a non-Git project root.

Do not scan outside the filesystem root.

Support Unicode paths and paths containing spaces.

## 6.3 Stable project identity

Use the following identity priority:

1. sanitized canonical Git remote;
2. Git common directory identity when no remote exists;
3. canonical project root path.

Sanitize remote URLs:

- remove username and password;
- remove tokens;
- remove query string;
- normalize SSH SCP syntax;
- lowercase hostname;
- remove trailing `.git`;
- never persist credentials.

Generate a stable project ID using SHA-256.

Do not use only the directory basename.

## 6.4 Git worktrees

Git worktrees belonging to the same repository share project memory by default.

Store observed roots as aliases.

Include branch in capture metadata, but do not create separate storage per branch in v0.1.

## 6.5 Monorepos

Default to Git root.

Support an optional override file:

```text
.herdr-logbook.toml
```

Example:

```toml
project_id = "company-platform-api"
display_name = "Platform API"
root = "services/api"
storage = "central"
```

Validate that overridden paths do not escape the expected repository boundary.

## 6.6 Project registry

Suggested record:

```toml
id = "p_..."
name = "example-project"
storage = "central"
store_path = "..."
first_seen = "2026-07-22T09:00:00Z"
last_seen = "2026-07-22T09:40:00Z"
remote_fingerprint = "github.com/org/repo"
default_root = "/path/to/repo"
roots = ["/path/to/repo", "/path/to/worktree"]
```

Never store remote credentials.

---

# 7. Herdr integration

## 7.1 Use Herdr-provided environment

Read:

```text
HERDR_BIN_PATH
HERDR_PLUGIN_ROOT
HERDR_PLUGIN_CONFIG_DIR
HERDR_PLUGIN_STATE_DIR
HERDR_PLUGIN_CONTEXT_JSON
HERDR_WORKSPACE_ID
HERDR_TAB_ID
HERDR_PANE_ID
```

The available context may contain:

- workspace ID;
- workspace label;
- workspace working directory;
- focused pane ID;
- focused pane working directory;
- worktree information;
- selected terminal text;
- clicked URLs;
- invocation source.

Use `HERDR_BIN_PATH` to call Herdr. Do not implement raw Unix-socket or Windows named-pipe clients in v0.1.

## 7.2 Mandatory compatibility spike

Before implementing the product UI, complete Phase 0:

1. create a minimal manifest;
2. build a tiny Go binary that prints context and paths;
3. open it using a plugin action and popup;
4. test Linux, macOS, Windows, and WSL;
5. test UTF-8 paths;
6. test paths with spaces;
7. test PowerShell 5.1 and PowerShell 7;
8. verify action-to-pane context propagation;
9. determine exact `plugin pane open` flags;
10. document findings in:

```text
docs/herdr-compatibility.md
```

Do not build production launch logic around guessed Herdr behavior.

## 7.3 Popup behavior

The default Hub should open as a popup to avoid modifying the current pane layout.

Target dimensions:

```text
width = "92%"
height = "86%"
```

Quick capture should use a smaller popup.

A persistent split can be considered later. Do not add split toggle, pane heartbeat, duplicate detection, and focus orchestration to v0.1 unless clearly necessary.

## 7.4 Windows

Relative executable paths in plugin panes may be unreliable on Windows. Resolve the binary through an absolute path based on `HERDR_PLUGIN_ROOT`.

Use platform-specific action or pane IDs when necessary.

Suggested logical IDs:

Unix:

```text
open
capture-project
capture-global
new-decision
capture-selection
```

Windows:

```text
open-windows
capture-project-windows
capture-global-windows
new-decision-windows
capture-selection-windows
```

Keep human-readable action titles identical where possible.

## 7.5 Keybindings

Do not assume universal default keys.

Provide:

```bash
herdr-logbook keybinds
```

It should print platform-correct TOML examples.

Example:

```toml
[[keys.command]]
key = "prefix+m"
type = "plugin_action"
command = "herdr-logbook.open"
description = "open project memory"
```

---

# 8. CLI contract

Implement:

```text
herdr-logbook tui
herdr-logbook capture
herdr-logbook decision
herdr-logbook init
herdr-logbook doctor
herdr-logbook keybinds
herdr-logbook paths
herdr-logbook index rebuild
herdr-logbook version
```

## 8.1 `tui`

```text
herdr-logbook tui
  --view now|project|global|all
  --project-root PATH
  --context-json JSON
```

## 8.2 `capture`

```text
herdr-logbook capture
  --scope project|global
  --text TEXT
  --stdin
  --selected
  --project-root PATH
  --branch BRANCH
  --source-cwd PATH
```

When neither `--text` nor `--stdin` is provided, open the capture TUI.

## 8.3 `decision`

```text
herdr-logbook decision
  --title TEXT
  --project-root PATH
```

Without a title, open the decision form.

## 8.4 `init`

```text
herdr-logbook init
  --storage central|repo
  --project-root PATH
```

Must be idempotent.

## 8.5 `doctor`

Display:

- plugin version;
- OS and architecture;
- Herdr binary and version;
- plugin root;
- config directory;
- state directory;
- parsed invocation context;
- selected working directory;
- resolved project root;
- project ID;
- sanitized remote fingerprint;
- current branch;
- selected storage mode;
- editor resolution;
- Glow availability;
- store permissions;
- cache state;
- actionable warnings.

Support:

```bash
herdr-logbook doctor --json
```

Never include selected terminal content or credentials.

## 8.6 Exit codes

Use stable exit codes:

```text
0 success
1 general runtime failure
2 invalid arguments or configuration
3 project context unavailable
4 data write or lock failure
5 Herdr integration failure
6 external editor failure
```

---

# 9. Capture format

## 9.1 Project/global capture

Append a timestamped section to the current monthly inbox.

Example:

```markdown
## 2026-07-22 14:32

- Branch: `feature/token-rotation`
- Source: `/workspace/api`

Token replay detection must invalidate the whole token family.
```

Do not include empty metadata rows.

## 9.2 Selected text capture

For multiline selected text:

```markdown
## 2026-07-22 14:35 — Terminal capture

- Branch: `feature/token-rotation`
- Source: `/workspace/api`

```text
<selected content>
```
```

Choose a safe fence length if the selected content already contains triple backticks.

For one-line selection, plain text or inline code is acceptable depending on content.

Default maximum selection size:

```text
512 KiB
```

If the selection exceeds the limit, show an explicit error or confirmation flow.

## 9.3 Concurrency

Capture must be safe when two commands write simultaneously.

Flow:

1. acquire target lock;
2. re-read file;
3. append;
4. write temporary file in same directory;
5. flush and close;
6. atomically replace destination;
7. release lock.

Use bounded lock timeouts.

---

# 10. Logbook Hub TUI

## 10.1 Wide layout

```text
┌ Scopes ─────────┬ Notes ─────────────────────┬ Preview ───────────────────────┐
│ ● Now           │ now.md                     │ # Now                          │
│   Project Inbox │ auth-notes.md              │                               │
│   Project Notes │ use-redis.md               │ ## Current task                │
│   Decisions     │                            │ ...                            │
│                 │                            │                               │
│   Global Inbox  │                            │                               │
│   All Notes     │                            │                               │
├─────────────────┴────────────────────────────┴────────────────────────────────┤
│ project: api · branch: feature/login · central store · / search · ? help    │
└──────────────────────────────────────────────────────────────────────────────┘
```

## 10.2 Responsive behavior

- at 110+ columns: three panes;
- at 70–109 columns: sidebar plus one content pane;
- below 70 columns: one active pane;
- use `Tab` and `Shift+Tab` to navigate panels;
- do not require horizontal scrolling;
- rerender Markdown based on current preview width.

## 10.3 Visual style

Use a restrained developer-tool aesthetic:

- semantic active-selection style;
- muted metadata;
- clear scope badges;
- subtle decision status;
- readable search highlights;
- dark/light terminal compatibility;
- no gradients;
- no excessive borders;
- no decorative rainbow palette.

## 10.4 Default keys

Navigation:

```text
j / k or arrows    move
h / l              change panel
Tab / Shift+Tab    next/previous panel
g / G              top/bottom
Enter              open/select
/                  search
Esc                clear search or close modal
?                  help
q                  close
```

Actions:

```text
c                  project capture
C                  global capture
n                  create project note
d                  create decision
e                  edit in external editor
r                  refresh
p                  switch/search project
v                  toggle list/preview on narrow terminal
```

Add a command palette on `Ctrl+P` only after core navigation works.

## 10.5 Empty states

Empty states must explain the next action.

Examples:

```text
No project notes yet.
Press n to create a note or c to capture something.
```

Do not show blank panels without explanation.

---

# 11. Search and index

## 11.1 Source of truth

Markdown files are canonical.

The index can always be deleted and rebuilt.

## 11.2 Initial design

Use an in-memory index plus a JSON cache.

Store:

- path;
- project ID;
- project display name;
- note type;
- title;
- tags;
- modified time;
- size;
- searchable content up to configured limit;
- content fingerprint.

Do not use SQLite in v0.1.

## 11.3 Startup

1. load cached index;
2. render Hub immediately;
3. scan registered stores asynchronously;
4. update changed entries;
5. remove deleted entries;
6. atomically replace cache.

Do not block the initial TUI render on a complete scan.

## 11.4 Ranking

Rank by:

1. exact title;
2. title prefix;
3. fuzzy title/path;
4. tag match;
5. content substring;
6. recency.

Ranking must be deterministic and unit-tested.

## 11.5 Limits

Defaults:

```text
indexed content per file: 256 KiB
preview file size: 2 MiB
follow symlinks: false
file extension: .md only
```

Avoid scanning hidden directories outside known memory roots.

## 11.6 Performance targets

Targets on a modern laptop:

```text
CLI capture: <150 ms
cached Hub first render: <200 ms
search response for 10,000 notes: <50 ms
full rescan of 10,000 small notes: <2 s
idle CPU: approximately zero
release binary: <30 MiB stripped
```

Do not publish these as claims until measured.

Add benchmarks for:

- search ranking;
- cache load;
- project resolution;
- scan speed;
- capture formatting.

---

# 12. Reliability and safety

## 12.1 Atomic writes

For file replacement:

1. create temporary file in destination directory;
2. write content;
3. flush;
4. close;
5. atomically rename;
6. preserve reasonable permissions.

Handle Windows replacement semantics explicitly.

## 12.2 Locking

Use per-project or per-target locks for capture and registry updates.

Do not hold locks while an external editor is open.

## 12.3 Recovery

- cache corruption must self-heal;
- partial writes must not replace valid data;
- existing non-empty scaffold files must not be overwritten;
- missing registered project roots should be shown as unavailable, not deleted silently;
- moved repositories should be recoverable by observing new roots with the same remote fingerprint.

## 12.4 Destructive actions

Prefer omitting deletion in v0.1 rather than implementing unsafe permanent deletion.

If delete is implemented later, move files into a local trash directory.

## 12.5 Path safety

Reject:

- traversal outside configured stores;
- empty sanitized filenames;
- Windows reserved device names;
- collisions on case-insensitive filesystems;
- note paths outside the active store.

Handle Unicode safely.

---

# 13. Security and privacy

Required behavior:

- no telemetry;
- no cloud;
- no AI API;
- no runtime network calls;
- no execution of commands from notes;
- no automatic Git operations;
- no selected-text contents in logs;
- no credentials from Git remotes in registry or diagnostics;
- explicit repository-local storage opt-in;
- size limits for selected content;
- safe argv process execution;
- no shell interpolation of note titles or editor paths.

Installation scripts may download the release binary and checksum from the repository’s GitHub Releases. Normal runtime must remain offline.

---

# 14. Configuration

Store config in:

```text
HERDR_PLUGIN_CONFIG_DIR/config.toml
```

Suggested default configuration:

```toml
version = 1

[storage]
project_mode = "central"
repo_directory = ".herdr/logbook"

[project]
root_strategy = "git"
share_across_worktrees = true

[editor]
command = []

[ui]
theme = "auto"
preview_style = "auto"
show_branch = true
popup_width = "92%"
popup_height = "86%"

[search]
max_index_file_bytes = 262144
max_preview_file_bytes = 2097152
follow_symlinks = false

[capture]
max_selection_bytes = 524288
include_branch = true
include_source_cwd = true
```

Requirements:

- no configuration required for first run;
- strict type validation;
- useful validation errors;
- unknown keys produce warnings;
- versioned migrations;
- backward compatibility;
- do not rewrite user configuration unless required for migration.

---

# 15. Distribution

## 15.1 Users must not need Go

Publish prebuilt binaries:

```text
linux_amd64
linux_arm64
darwin_amd64
darwin_arm64
windows_amd64
windows_arm64
```

Release assets:

```text
herdr-logbook_<version>_linux_amd64.tar.gz
herdr-logbook_<version>_linux_arm64.tar.gz
herdr-logbook_<version>_darwin_amd64.tar.gz
herdr-logbook_<version>_darwin_arm64.tar.gz
herdr-logbook_<version>_windows_amd64.zip
herdr-logbook_<version>_windows_arm64.zip
checksums.txt
```

## 15.2 Herdr build scripts

Use platform-specific build/install commands in the manifest.

Unix:

```toml
[[build]]
platforms = ["linux", "macos"]
command = ["bash", "scripts/install.sh"]
```

Windows:

```toml
[[build]]
platforms = ["windows"]
command = [
  "powershell",
  "-NoProfile",
  "-ExecutionPolicy",
  "Bypass",
  "-File",
  "scripts/install.ps1"
]
```

Install scripts must:

1. read plugin version;
2. detect OS and architecture;
3. download the exact matching release;
4. download checksum file;
5. verify SHA-256;
6. extract into `bin/`;
7. set executable permission on Unix;
8. fail with a useful error.

Do not download an unpinned `latest` binary.

## 15.3 Local development

Unix:

```bash
go build -o bin/herdr-logbook ./cmd/herdr-logbook
herdr plugin link "$(pwd)"
```

Windows:

```powershell
go build -o bin\herdr-logbook.exe .\cmd\herdr-logbook
herdr plugin link (Get-Location)
```

Remember that `herdr plugin link` does not run build commands.

---

# 16. Testing

## 16.1 Unit tests

Required areas:

- context precedence;
- Git root resolution;
- non-Git project resolution;
- worktree identity;
- remote sanitization;
- stable hashing;
- Windows reserved names;
- path traversal;
- filename slugging;
- capture formatting;
- safe Markdown fencing;
- title extraction;
- tag extraction;
- search ranking;
- config validation;
- config migration;
- editor resolution;
- cache corruption;
- atomic write failure stages.

## 16.2 Integration tests

Use temporary directories and temporary Git repositories.

Cover:

- HTTPS remote;
- SSH remote;
- remote with credentials;
- repository without remote;
- non-Git directory;
- Git worktree;
- moved repository;
- central store;
- repo-local store;
- read-only repository;
- unavailable root;
- paths with spaces;
- Unicode paths;
- concurrent captures;
- external editor modification;
- simultaneous index update and capture.

## 16.3 TUI tests

Test Bubble Tea update behavior directly.

Use limited golden tests for:

- wide layout;
- narrow layout;
- empty state;
- search result;
- error modal;
- preview width.

Do not create a large fragile snapshot suite.

## 16.4 CI matrix

Run on:

- Ubuntu latest;
- macOS latest;
- Windows latest.

Checks:

```bash
go test ./...
go test -race ./...
go vet ./...
staticcheck ./...
govulncheck ./...
```

Also validate:

- shell syntax;
- PowerShell scripts;
- manifest consistency;
- release asset names;
- installer/checksum workflow;
- binary `version` smoke test.

## 16.5 Manual compatibility

Before v0.1.0 test:

- Linux Herdr;
- macOS Intel;
- macOS Apple Silicon;
- Windows Terminal;
- WSL 2;
- PowerShell 5.1;
- PowerShell 7;
- Bash;
- Zsh;
- 80x24 terminal;
- dark terminal;
- light terminal;
- project paths containing spaces;
- Czech and other non-ASCII characters.

---

# 17. Implementation phases

Complete phases in order.

## Phase 0 — Herdr compatibility spike

Deliver:

- minimal plugin manifest;
- diagnostic Go binary;
- context dump;
- popup action;
- verified launcher per platform;
- `docs/herdr-compatibility.md`.

Do not proceed with final launch architecture until this works.

## Phase 1 — Core domain

Implement:

- config;
- Herdr context parsing;
- project resolution;
- remote sanitization;
- project identity;
- registry;
- storage paths;
- central scaffold;
- repo-local initialization;
- atomic writes;
- locks;
- `paths`;
- `doctor`.

## Phase 2 — Capture

Implement:

- project capture;
- global capture;
- selected-text capture;
- monthly inboxes;
- metadata;
- capture textarea;
- size limits;
- concurrent write safety.

## Phase 3 — Read-only Hub

Implement:

- Bubble Tea shell;
- responsive layout;
- scopes;
- note list;
- `now.md`;
- inboxes;
- decisions;
- Glamour preview;
- help;
- empty states;
- asynchronous loading.

## Phase 4 — Search

Implement:

- project registry scan;
- cache;
- refresh;
- ranking;
- filters;
- matched snippets;
- `index rebuild`;
- performance benchmarks.

## Phase 5 — Authoring

Implement:

- create note;
- create decision;
- safe filename slugging;
- external editor;
- refresh after editor exit;
- optional external Glow.

## Phase 6 — Production Herdr integration

Implement:

- final manifest;
- platform-specific actions;
- popup launch;
- action-to-pane context forwarding;
- keybind generator;
- clean exits;
- useful command errors;
- Windows absolute paths.

## Phase 7 — Hardening

Implement:

- fuzz tests;
- race tests;
- recovery paths;
- read-only failures;
- invalid context;
- moved repositories;
- case-insensitive collisions;
- corruption recovery;
- diagnostics.

## Phase 8 — Release

Implement:

- release workflow;
- checksums;
- install scripts;
- changelog;
- security documentation;
- contribution guide;
- marketplace metadata;
- screenshots or terminal recording;
- fresh-profile installation test.

---

# 18. Definition of done for v0.1.0

Do not call v0.1.0 complete until all are true:

- users can install without Go;
- tagged release assets exist for all supported targets;
- installation verifies checksums;
- Hub opens from a Herdr action;
- active project is resolved correctly;
- `now.md` is created safely and displayed first;
- project capture works;
- global capture works;
- selected-text capture works;
- decision creation works;
- global cross-project search works;
- Markdown preview uses Glamour;
- full editing uses an external editor;
- central storage is default;
- repo-local storage is explicit;
- concurrent writes cannot corrupt notes;
- cache corruption is recoverable;
- `doctor` is useful;
- no credentials leak;
- no telemetry exists;
- normal runtime makes no network requests;
- Windows paths with spaces work;
- Unicode paths work;
- narrow terminals remain usable;
- README claims are accurate;
- CI passes on Linux, macOS, and Windows;
- upgrade testing preserves data and configuration;
- uninstalling the plugin does not silently delete user notes.

---

# 19. README requirements

Create a professional `README.md` containing:

1. concise product description;
2. clear explanation of the problem;
3. core workflows;
4. terminal UI example;
5. storage explanation;
6. central vs repo-local mode;
7. project resolution;
8. worktree behavior;
9. Markdown/Glamour explanation;
10. external editor behavior;
11. installation;
12. local development;
13. platform-specific keybinding examples;
14. commands;
15. configuration;
16. privacy and data safety;
17. limitations and non-goals;
18. roadmap;
19. license;
20. screenshots or terminal recording before public release.

Avoid phrases such as:

- “AI second brain”;
- “revolutionary”;
- “Obsidian for terminal”;
- “complete knowledge platform.”

Use precise wording:

- project-aware developer memory;
- quick capture;
- global search;
- decision records;
- local Markdown files;
- Glow-style Markdown preview.

---

# 20. Suggested README UI example

Use a representation similar to:

```text
┌ Scopes ─────────┬ Notes ─────────────────────┬ Preview ───────────────────────┐
│ ● Now           │ now.md                     │ # Now                          │
│   Project Inbox │ auth-notes.md              │                               │
│   Project Notes │ use-redis.md               │ ## Current task                │
│   Decisions     │                            │ Implement token rotation.      │
│                 │                            │                               │
│   Global Inbox  │                            │ ## Next steps                  │
│   All Notes     │                            │ - [ ] Add replay detection.    │
├─────────────────┴────────────────────────────┴────────────────────────────────┤
│ api-gateway · feature/token-rotation · central store · / search · ? help    │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

# 21. Implementation rules for the AI agent

Follow these rules:

1. Read this entire document before writing code.
2. Start with Phase 0.
3. Do not guess undocumented Herdr behavior.
4. Test every OS-specific launcher assumption.
5. Keep platform scripts minimal.
6. Keep business logic inside Go.
7. Do not add AI features.
8. Do not add cloud or sync.
9. Do not build a custom Vim mode.
10. Do not silently modify repositories.
11. Do not add SQLite without benchmark evidence.
12. Do not store credentials from remotes.
13. Do not log selected text.
14. Use atomic writes.
15. Use locking for concurrent capture.
16. Preserve user data during upgrades.
17. Treat the search cache as rebuildable.
18. Document architectural deviations in `docs/decisions/`.
19. Add tests with every phase.
20. Update README whenever behavior changes.
21. Keep dependencies pinned and justified.
22. Do not claim cross-platform support until tested.
23. Prefer a safe omitted feature over an unsafe partial implementation.
24. Run the actual test suite before declaring a phase complete.
25. Report unresolved Herdr API limitations explicitly.

Recommended commit order:

```text
1. bootstrap and compatibility spike
2. project resolution and storage
3. capture workflows
4. Hub and Markdown preview
5. index and search
6. authoring and editor integration
7. Herdr actions and launchers
8. hardening
9. packaging and documentation
```

At the end of each phase, provide:

- files changed;
- functionality completed;
- tests run;
- test results;
- known limitations;
- next phase;
- any deviation from this specification.

---

# 22. Future roadmap

Only consider after stable v0.1 usage:

- inbox item promotion;
- archive/trash;
- saved searches;
- templates;
- project aliases UI;
- optional persistent split pane;
- link capture;
- branch filters;
- `[[wikilinks]]`;
- backlinks;
- Markdown import/export helpers;
- shared repo documentation safeguards.

Explicitly deferred until real user demand:

- AI;
- sync;
- collaboration;
- browser UI;
- mobile UI;
- graph visualization.

---

# Final instruction

Implement this as a focused, fast, local developer tool.

The most important product loop is:

```text
detect project
→ capture context
→ persist safely
→ reopen project
→ immediately recover context
→ search it later
```

Optimize that loop before adding secondary features.
