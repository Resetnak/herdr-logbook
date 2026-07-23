<div align="center">

# 📓 Herdr Logbook

**Local, offline, Markdown-first working memory for developers using Herdr.**

Keep active tasks, quick captures, architectural decisions, and project notes right in your terminal.  
No proprietary formats, no cloud locks, zero telemetry.

<br>

[![CI](https://github.com/Resetnak/herdr-logbook/actions/workflows/ci.yml/badge.svg)](https://github.com/Resetnak/herdr-logbook/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platforms](https://img.shields.io/badge/platforms-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)](docs/herdr-compatibility.md)
[![Herdr](https://img.shields.io/badge/Herdr-%E2%89%A50.7.0-2088FF)](herdr-plugin.toml)
[![Status](https://img.shields.io/badge/status-pre--release-orange)](CHANGELOG.md)

<br>

**English** · [Čeština](README.cs.md)

[Features](#-key-features) · [Quick Start](#-quick-start) · [Keyboard Shortcuts](#-keyboard-shortcuts) · [Installation](#-installation) · [Configuration](#-configuration) · [Privacy](#-privacy--security) · [Contributing](CONTRIBUTING.md)

</div>

---

## 💡 Why Herdr Logbook?

The context around a coding task often outlives the terminal session that created it. **Herdr Logbook** captures that context right inside your terminal — stored as clean, standard Markdown files that remain yours forever, with or without the plugin.

```text
┌ Scopes ─────────┬ Notes ─────────────────────┬ Preview ───────────────────────┐
│ ● Now           │ now.md                     │ # Now                         │
│ Project Inbox   │ auth-notes.md              │ ## Current task               │
│ Project Notes   │ architecture-adr.md        │ Implement token rotation.     │
│ Decisions       │                            │                               │
│ Global Inbox    │                            │ ## Next steps                  │
│ All Notes       │                            │ - [ ] Add replay detection.   │
└─────────────────┴────────────────────────────┴────────────────────────────────┘
 api-gateway · feature/token-rotation · central store · / search · ? help
```

---

## ✨ Key Features

- **📓 Markdown-First & Canonical**: Plain `.md` files in your storage. Disposable, self-healing search index.
- **🌿 Git & Repository Aware**: Automatically scopes notes by workspace, Git worktree, or current folder.
- **⚡ Responsive Terminal UI**: Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Adapts from 3 panes to a single pane on narrow terminals (<70 cols).
- **⚖️ Architectural Decisions (ADR)**: Built-in templates for logging technical choices and consequences (`d`).
- **🔍 Instant Fuzzy Search**: Real-time cross-project search (`/`) or project-filtered search (`p`).
- **📝 Bring Your Own Editor**: Delegates full editing to your preferred `$EDITOR` (`nvim`, `vim`, `nano`, `code`).
- **🔒 100% Offline & Private**: Zero telemetry, no cloud sync, no background Git operations, no AI lock-in.

---

## 🎮 Keyboard Shortcuts

### 🧭 Navigation & Panels
| Shortcut | Action |
| :--- | :--- |
| `Tab` / `Shift+Tab` or `h` / `l` | Switch active panel (`Scopes` → `Notes` → `Preview`) |
| `j` / `k` or `↑` / `↓` | Navigate items in list |
| `g` / `G` | Jump to top / bottom of list |
| `Enter` / `v` | Open note in Markdown preview |
| `/` | Fuzzy search notes across all projects |
| `p` | Filter search by project name |

### ⚡ Actions
| Shortcut | Action |
| :--- | :--- |
| `c` / `C` | Quick capture (project or global inbox) |
| `n` | Create new project note |
| `d` | Record architectural decision (ADR) |
| `e` | Edit selected note in `$EDITOR` (`vi` / `nvim`) |
| `r` | Refresh note index |
| `?` | Toggle interactive Onboarding & Help screen |
| `q` | Quit Logbook |

### 📝 Capture & Authoring Modal
| Shortcut | Action |
| :--- | :--- |
| `Ctrl+S` | **Save note** (saves directly without leaving TUI) |
| `Ctrl+E` | **Save & edit** (saves note and launches `$EDITOR`) |
| `Esc` | Cancel capture |

> 💡 **Markdown Hint**: Notes support standard Markdown syntax (`# Heading`, `**bold**`, `- list`, `` `code` ``, `#tag`).

---

## 📦 Installation

Herdr Logbook requires [Herdr](https://github.com/Resetnak/herdr) ≥ 0.7.0.

> ⚡ **No Go Compiler Required for End Users**: The plugin installation automatically downloads SHA-256 checksum-verified precompiled binaries for Linux, macOS, and Windows (`amd64` & `arm64`) directly from GitHub Releases via `scripts/install.sh` / `scripts/install.ps1`.

### Option 1: Via Herdr Plugin Manager (Recommended)

Run inside Herdr or your terminal:

```bash
herdr plugin install Resetnak/herdr-logbook
```

Herdr will fetch the repository, run the installer script, verify the release binary checksum, and register the plugin automatically.

---

### Option 2: Quick Script Installer (Standalone / Manual)

#### 🍏 macOS & 🐧 Linux / WSL
```bash
curl -fsSL https://raw.githubusercontent.com/Resetnak/herdr-logbook/main/scripts/install.sh | bash
```

#### 🪟 Windows PowerShell
```powershell
irm https://raw.githubusercontent.com/Resetnak/herdr-logbook/main/scripts/install.ps1 | iex
```

---

### Option 3: Direct Binary Download (GitHub Releases)

Download precompiled standalone binaries (with SHA-256 checksum verification) from [GitHub Releases](https://github.com/Resetnak/herdr-logbook/releases):

| OS | Architecture | Archive |
| :--- | :--- | :--- |
| **macOS** | Apple Silicon (`arm64`) / Intel (`amd64`) | `.tar.gz` |
| **Linux** | `amd64` / `arm64` | `.tar.gz` |
| **Windows** | `amd64` / `arm64` | `.zip` |

---

### Option 4: Build from Source (For Developers)

Building from source requires **Go ≥ 1.25**.

#### 1. Install Go (if not already installed)
- **macOS (via Homebrew)**: `brew install go`
- **Linux (Ubuntu/Debian)**: `sudo apt update && sudo apt install golang`
- **Windows (via Winget)**: `winget install GoLang.Go`

#### 2. Clone & Build
```bash
# macOS / Linux
git clone https://github.com/Resetnak/herdr-logbook.git
cd herdr-logbook
mkdir -p bin
go build -o bin/herdr-logbook ./cmd/herdr-logbook
herdr plugin link "$(pwd)" --enabled
```

```powershell
# Windows PowerShell
git clone https://github.com/Resetnak/herdr-logbook.git
Set-Location herdr-logbook
New-Item -ItemType Directory -Force bin | Out-Null
go build -o bin\herdr-logbook.exe .\cmd\herdr-logbook
herdr plugin link (Get-Location) --enabled
```

---

## ⚙️ Configuration

Create or edit your config at `$(herdr plugin config-dir herdr-logbook)/config.toml` (see [config.example.toml](config.example.toml)):

```toml
version = 1

[editor]
# External editor command (e.g. ["nvim"], ["vim"], ["nano"], ["code", "--wait"])
command = ["nvim"]

[ui]
# TUI Color Theme: "auto", "dracula", "tokyo-night", "nord", or "default"
theme = "dracula"
# Markdown Preview Style: "auto", "dark", "light", or "notty"
preview_style = "dark"
# Show Git branch name in status bar
show_branch = true
# Width of Scopes pane in columns (default: 24)
scope_width = 24
# Default view tab on launch: "now", "project", "global", or "all"
default_view = "now"

[storage]
# Storage mode: "central" (state directory) or "repo" (.herdr/logbook in project)
project_mode = "central"
```

### Bind a Shortcut in Herdr

Add the action to your Herdr configuration (`~/.config/herdr/config.toml`):

```toml
[[keys.command]]
key = "prefix+m"
type = "plugin_action"
command = "herdr-logbook.open"
description = "Herdr Logbook"
```

Reload Herdr configuration:
```bash
herdr config check
herdr server reload-config
```

---

## 📁 Storage Structure

By default, notes are stored centrally in your state directory (`central` mode), keeping your git working trees clean:

```text
$HERDR_PLUGIN_STATE_DIR/
├── store/projects/p_<sha256>/
│   ├── now.md
│   ├── inbox/
│   ├── notes/
│   └── decisions/
├── store/global/
├── registry/projects.toml
└── cache/index-v1.json
```

---

## 🔒 Privacy & Security

- **Zero Telemetry**: No tracking, network calls, or analytics.
- **Data Protection**: Git credentials are automatically sanitized from remote URLs.
- **Safe I/O**: All file modifications use atomic writes with file locks and fsync.

---

## 🤝 Contributing

Contributions, bug reports, and feature requests are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for local setup and testing guidelines.

## 📄 License

[MIT](LICENSE) © 2026 Alexandr Rešetňak
