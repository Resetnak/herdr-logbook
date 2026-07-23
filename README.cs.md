<div align="center">

# 📓 Herdr Logbook

**Lokální, offline pracovní paměť pro vývojáře v Herdru — postavená na Markdownu.**

Držte své aktivní úkoly, rychlé poznámky, architektonická rozhodnutí a zápisky přímo v terminálu.  
Žádné proprietární formáty, žádný cloud, žádná telemetrie.

<br>

[![CI](https://github.com/Resetnak/herdr-logbook/actions/workflows/ci.yml/badge.svg)](https://github.com/Resetnak/herdr-logbook/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![Licence: MIT](https://img.shields.io/badge/Licence-MIT-green.svg)](LICENSE)
[![Platformy](https://img.shields.io/badge/platformy-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)](docs/herdr-compatibility.md)
[![Herdr](https://img.shields.io/badge/Herdr-%E2%89%A50.7.0-2088FF)](herdr-plugin.toml)
[![Stav](https://img.shields.io/badge/stav-pre--release-orange)](CHANGELOG.md)

<br>

[English](README.md) · **Čeština**

[Vlastnosti](#-hlavní-vlastnosti) · [Klávesové zkratky](#-klávesové-zkratky) · [Instalace](#-instalace) · [Konfigurace](#-konfigurace) · [Soukromí](#-soukromí-a-bezpečnost) · [Přispívání](CONTRIBUTING.md)

</div>

---

## 💡 Proč Herdr Logbook?

Užitečný kontext kolem vývojového úkolu často přežije terminálovou relaci, ve které vznikl. **Herdr Logbook** drží tento kontext na dosah jedné zkratky přímo v terminálu — uložený v čistém Markdownu, který vám zůstane navždy přístupný s pluginem i bez něj.

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

## ✨ Hlavní vlastnosti

- **📓 Markdown na prvním místě**: Poznámky jsou čisté `.md` soubory. Vyhledávací index je jednorázový a sám se obnovuje.
- **🌿 Vnímání projektů a Git**: Poznámky se automaticky organizují podle repozitáře, Git worktree nebo adresáře.
- **⚡ Responzivní TUI rozhraní**: Postavené na [Bubble Tea](https://github.com/charmbracelet/bubbletea). Na úzkých terminálech (<70 sloupců) se automaticky přepne na jednosloupcový režim.
- **⚖️ Architektonická rozhodnutí (ADR)**: Vestavěná šablona pro záznam technických rozhodnutí a důsledků (`d`).
- **🔍 Okamžité vyhledávání**: Bleskové fuzzy vyhledávání napříč všemi projekty (`/`) nebo filtrování podle projektu (`p`).
- **📝 Vlastní editor**: Úpravy souborů deleguje na váš oblíbený `$EDITOR` (`nvim`, `vim`, `nano`, `code`).
- **🔒 100% Offline & Soukromí**: Žádná telemetrie, žádný cloud, žádné automatické Git operace ani závislost na AI.

---

## 🎮 Klávesové zkratky

### 🧭 Navigace & Panely
| Klávesa | Akce |
| :--- | :--- |
| `Tab` / `Shift+Tab` nebo `h` / `l` | Přepínání panelů (`Scopes` → `Notes` → `Preview`) |
| `j` / `k` nebo `↑` / `↓` | Pohyb v seznamu |
| `g` / `G` | Skok na začátek / konec seznamu |
| `Enter` / `v` | Otevřít náhled Markdownu |
| `/` | Fuzzy vyhledávání napříč všemi projekty |
| `p` | Filtrovat vyhledávání podle názvu projektu |

### ⚡ Akce
| Klávesa | Akce |
| :--- | :--- |
| `c` / `C` | Rychlé zachycení (projektový / globální inbox) |
| `n` | Vytvořit novou projektovou poznámku |
| `d` | Zaznamenat architektonické rozhodnutí (ADR) |
| `e` | Otevřít vybranou poznámku v `$EDITOR` (`vi` / `nvim`) |
| `r` | Obnovit index poznámek |
| `?` | Zobrazit interaktivní nápovědu a Onboarding |
| `q` | Zavřít Logbook |

### 📝 Modální okno pro psaní
| Klávesa | Akce |
| :--- | :--- |
| `Ctrl+S` | **Uložit poznámku** (uloží přímo bez opuštění TUI) |
| `Ctrl+E` | **Uložit & upravit** (uloží poznámku a spustí `$EDITOR`) |
| `Esc` | Zrušit |

> 💡 **Markdown Tip**: Poznámky podporují běžnou syntaxi (`# Nadpis`, `**tučně**`, `- seznam`, `` `kód` ``, `#tag`).

---

## 📦 Instalace

Herdr Logbook vyžaduje [Herdr](https://github.com/Resetnak/herdr) ≥ 0.7.0 a Go ≥ 1.25.

### 🍏 macOS / 🐧 Linux / 💻 WSL

```bash
git clone https://github.com/Resetnak/herdr-logbook.git
cd herdr-logbook
mkdir -p bin
go build -o bin/herdr-logbook ./cmd/herdr-logbook
herdr plugin link "$(pwd)" --enabled
herdr plugin action invoke open --plugin herdr-logbook
```

### 🪟 Windows PowerShell

```powershell
git clone https://github.com/Resetnak/herdr-logbook.git
Set-Location herdr-logbook
New-Item -ItemType Directory -Force bin | Out-Null
go build -o bin\herdr-logbook.exe .\cmd\herdr-logbook
herdr plugin link (Get-Location) --enabled
herdr plugin action invoke open-windows --plugin herdr-logbook
```

---

## ⚙️ Konfigurace

Vytvořte nebo upravte konfiguraci v `$(herdr plugin config-dir herdr-logbook)/config.toml`:

```toml
[editor]
command = ["nvim"]
```

### Nastavení zkratky v Herdru

Přidejte akci do `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+m"
type = "plugin_action"
command = "herdr-logbook.open"
description = "Herdr Logbook"
```

Znovu načtěte konfiguraci Herdru:
```bash
herdr config check
herdr server reload-config
```

---

## 📁 Struktura Úložiště

Soubory jsou výchozím způsobem ukládány centrálně (`central` režim), takže vaše Git repozitáře zůstávají čisté:

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

## 🔒 Soukromí a Bezpečnost

- **Nulová Telemetrie**: Žádné sledování, síťové požadavky ani analytika.
- **Ochrana Dat**: Git přihlašovací údaje se automaticky odstraňují z URL adres.
- **Bezpečný Zápis**: Všechny operace používají atomický zápis se zámky souborů a `fsync`.

---

## 🤝 Přispívání

Návrhy, hlášení chyb a PR jsou vítány! Podívejte se do [CONTRIBUTING.md](CONTRIBUTING.md) pro lokální nastavení a instrukce k testování.

## 📄 Licence

[MIT](LICENSE) © 2026 Alexandr Rešetňak
