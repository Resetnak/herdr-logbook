<div align="center">

# 📓 Herdr Logbook

**Pracovní paměť vašeho terminálu — offline, Markdown, vaše.**

Aktivní úkoly, rychlé poznámky a architektonická rozhodnutí, uložené jako čisté
`.md` soubory, které si můžete grepnout, gitnout a přečíst i za 20 let.

*Žádný cloud. Žádná AI. Žádná telemetrie. Žádný SQLite. Záměrně.*

<br>

[![CI](https://github.com/Resetnak/herdr-logbook/actions/workflows/ci.yml/badge.svg)](https://github.com/Resetnak/herdr-logbook/actions/workflows/ci.yml)
[![Pokrytí](https://img.shields.io/badge/pokrytí-90%25-brightgreen)](https://github.com/Resetnak/herdr-logbook/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![Licence: MIT](https://img.shields.io/badge/Licence-MIT-green.svg)](LICENSE)
[![Testováno jen na macOS](https://img.shields.io/badge/testováno-jen%20macOS-orange)](docs/herdr-compatibility.md)
[![Herdr](https://img.shields.io/badge/Herdr-%E2%89%A50.7.0-2088FF)](herdr-plugin.toml)
[![Stav](https://img.shields.io/badge/stav-pre--release-orange)](CHANGELOG.md)

<br>

[English](README.md) · **Čeština**

[Rychlý start](#-rychlý-start) · [Vlastnosti](#-hlavní-vlastnosti) · [Srovnání](#-srovnání-s-jinými-nástroji) · [Klávesové zkratky](#-klávesové-zkratky) · [Instalace](#-instalace) · [Konfigurace](#-konfigurace) · [Soukromí](#-soukromí-a-bezpečnost) · [Přispívání](CONTRIBUTING.md)

</div>

<div align="center">
  <img src="assets/demo.gif" alt="Claude Code běžící v panelu Herdru; po stisku Ctrl+B a M se vedle otevře Logbook Hub" width="820">

  <sub>Claude Code pracuje v repozitáři — stiskněte <code>Ctrl+B</code> a poté <code>m</code> a vedle agenta se otevře Logbook Hub: inbox, rozhodnutí, vyhledávání napříč projekty. Vyrenderováno z <a href="cassette.tape">cassette.tape</a>.</sub>
</div>

> ⚠️ **Testováno pouze na macOS (arm64).** Binárky pro Linux a Windows se kompilují křížově a procházejí jednotkovými testy v CI, ale plugin **nebyl** na těchto platformách ověřen proti skutečnému hostiteli Herdr. Používáte je na vlastní riziko. Viz [matice kompatibility](docs/herdr-compatibility.md).
>
> Používáte Herdr na Linuxu nebo Windows? Krátká zpráva, že plugin funguje (nebo nefunguje), je nejrychlejší cesta k ověření vaší platformy — [otevřete issue](https://github.com/Resetnak/herdr-logbook/issues).

---

## 💡 Proč Herdr Logbook?

Poznámky kolem vývojového úkolu by měly přežít nástroj, který je zachytil. **Herdr Logbook** drží tento kontext v terminálu jako čistý, standardní Markdown — soubory, které vám zůstanou s pluginem i bez něj, s Herdrem i bez něj, ať už tento repozitář udržuji, nebo ne.

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

## 🚀 Rychlý start

1. **Nainstalujte Herdr** (hostitelská terminálová aplikace): `brew install herdr` — viz [herdr.dev](https://herdr.dev).
2. **Nainstalujte tento plugin**: `herdr plugin install Resetnak/herdr-logbook`
3. **Otevřete Hub**: spusťte v Herdru akci *Open Logbook Hub* — nebo si [nastavte zkratku](#nastavení-zkratky-v-herdru) třeba `prefix+m`.
4. **Zachyťte**: `t` pro nastavení aktuálního úkolu, `c` pro poznámku, `d` pro rozhodnutí, `/` pro hledání. Vše se ukládá jako čisté `.md` soubory, které vlastníte — žádný vendor lock-in.

---

## ✨ Hlavní vlastnosti

- **📓 Markdown na prvním místě**: Poznámky jsou čisté `.md` soubory. Vyhledávací index je jednorázový a sám se obnovuje.
- **🌿 Vnímání projektů a Git**: Poznámky se automaticky organizují podle repozitáře, Git worktree nebo adresáře.
- **⚡ Responzivní TUI rozhraní**: Postavené na [Bubble Tea](https://github.com/charmbracelet/bubbletea). Na úzkých terminálech (<70 sloupců) se automaticky přepne na jednosloupcový režim.
- **🎯 Aktuální úkol a deník práce zdarma**: `t` (nebo `herdr-logbook now "…"`) nastaví aktuální úkol v `now.md`. Při každém přepnutí se předchozí úkol založí do měsíčního inboxu — deník se píše sám.
- **⚖️ Architektonická rozhodnutí (ADR)**: Vestavěná šablona pro záznam technických rozhodnutí a důsledků (`d`).
- **🔍 Okamžité vyhledávání**: Bleskové fuzzy vyhledávání napříč všemi projekty (`/`) nebo filtrování podle projektu (`p`).
- **📝 Vlastní editor**: Úpravy souborů deleguje na váš oblíbený `$EDITOR` (`nvim`, `vim`, `nano`, `code`).
- **🔒 100% Offline & Soukromí**: Žádná telemetrie, žádný cloud, žádné automatické Git operace ani závislost na AI.

---

## 🧭 Srovnání s jinými nástroji

Herdr Logbook nenahrazuje vaši znalostní bázi — zachycuje pracovní kontext *kolem vývojového úkolu*, automaticky navázaný na projekt, ve kterém právě jste.

| Nástroj | V čem je skvělý | V čem se Logbook liší |
| :--- | :--- | :--- |
| **Obsidian / Logseq** | Dlouhodobá znalostní báze | Logbook žije v terminálu hned vedle shellu a poznámky automaticky organizuje podle repozitáře/worktree — žádná aplikace, žádný vault |
| **zk / nb** | Obecné CLI poznámky | Logbook je zaměřený na úkoly: aktivní `now.md`, ADR šablony a panely/akce v Herdru — ne obecný zápisník |
| **`TODO.md` v repozitáři** | Nulové nástroje | Logbook drží pracovní stromy čisté (centrální úložiště), hledá napříč všemi projekty a přežije přepínání větví |

Vše je čistý Markdown, takže na kterýkoli z těchto nástrojů můžete kdykoli přejít — nebo je kombinovat.

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
| `t` | Nastavit aktuální úkol v `now.md` (předchozí se archivuje do inboxu) |
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

Herdr Logbook vyžaduje [Herdr](https://herdr.dev) ≥ 0.7.0 — nainstalujete jej přes `brew install herdr` (viz [herdr.dev](https://herdr.dev)).

> ⚡ **Není potřeba kompilátor Go**: Instalátor stáhne z GitHub Releases předkompilovanou binárku pro vaši platformu a ověří její SHA-256 kontrolní součet (skripty `scripts/install.sh` / `scripts/install.ps1`). Binárky se sestavují pro Linux, macOS i Windows (`amd64` a `arm64`), ale jen macOS (arm64) je ověřen proti skutečnému hostiteli Herdr — viz stav platforem níže.

> ⚠️ **Stav platforem — pouze macOS**: Tento plugin byl testován **pouze na macOS (arm64)**. Jednotkové testy sice běží v CI na Linuxu, macOS i Windows, ale samotná integrace pluginu s Herdrem (panely a akce) **nebyla** na Linuxu ani Windows testována. Tyto balíčky jsou poskytovány „tak jak jsou". Viz [matice kompatibility](docs/herdr-compatibility.md).

### Možnost 1: Přes Herdr Plugin Manager (Doporučeno)

Spusťte v Herdru nebo v terminálu:

```bash
herdr plugin install Resetnak/herdr-logbook
```

Herdr si stáhne repozitář, spustí instalační skript, ověří SHA-256 kontrolní součet sestavené binárky a automaticky plugin zaregistruje.

---

### Možnost 2: Rychlý skriptový instalátor (Samostatná / Ruční instalace)

#### 🍏 macOS & 🐧 Linux / WSL
```bash
curl -fsSL https://raw.githubusercontent.com/Resetnak/herdr-logbook/main/scripts/install.sh | bash
```

#### 🪟 Windows PowerShell
```powershell
irm https://raw.githubusercontent.com/Resetnak/herdr-logbook/main/scripts/install.ps1 | iex
```

---

### Možnost 3: Přímé stažení binárního balíčku (GitHub Releases)

Stáhněte si samostatné předkompilované balíčky (s ověřeným SHA-256 kontrolním součtem) přímo ze stránek [GitHub Releases](https://github.com/Resetnak/herdr-logbook/releases):

| OS | Architektura | Archív |
| :--- | :--- | :--- |
| **macOS** | Apple Silicon (`arm64`) / Intel (`amd64`) | `.tar.gz` |
| **Linux** | `amd64` / `arm64` | `.tar.gz` |
| **Windows** | `amd64` / `arm64` | `.zip` |

---

### Možnost 4: Kompilace ze zdrojového kódu (Pro vývojáře)

Kompilace ze zdrojového kódu vyžaduje **Go ≥ 1.25**.

Chcete jen samostatné CLI (bez registrace pluginu v Herdru)?

```bash
go install github.com/Resetnak/herdr-logbook/cmd/herdr-logbook@latest
```

Pro registraci jako Herdr plugin místo toho naklonujte a nalinkujte:

#### 1. Instalace Go (pokud ještě nemáte)
- **macOS (přes Homebrew)**: `brew install go`
- **Linux (Ubuntu/Debian)**: `sudo apt update && sudo apt install golang`
- **Windows (přes Winget)**: `winget install GoLang.Go`

#### 2. Klonování a kompilace
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

### Aktualizace

Herdr nemá příkaz `plugin update` — aktualizace se dělá opětovným spuštěním instalace. Herdr znovu stáhne repozitář a vezme si release binárku odpovídající verzi v `herdr-plugin.toml`.

```bash
herdr plugin install Resetnak/herdr-logbook              # nejnovější main
herdr plugin install Resetnak/herdr-logbook --ref v0.0.4 # konkrétní vydání
```

> ⚠️ Pokud jste instalovali Možností 4, tento příkaz skončí chybou `plugin herdr-logbook is already linked from a local path`. Lokální link má vždy přednost před GitHubem; nejdřív spusťte `herdr plugin unlink herdr-logbook`, nebo místo toho aktualizujte svůj checkout (viz níže).

Pokud jste použili skriptový instalátor (Možnost 2), spusťte znovu tentýž řádek `curl … | bash`, resp. `irm … | iex`.

Pokud máte přilinkovaný zdrojový checkout (Možnost 4), stáhněte změny a přebuildujte. Herdr čte skripty pluginu přímo z vašeho pracovního stromu, ale binárku si musíte zkompilovat sami:

```bash
git pull && go build -o bin/herdr-logbook ./cmd/herdr-logbook
```

---

## ⚙️ Konfigurace

Vytvořte nebo upravte konfiguraci v `$(herdr plugin config-dir herdr-logbook)/config.toml` (příklad v [config.example.toml](config.example.toml)):

```toml
version = 1

[editor]
# Příkaz externího editoru (např. ["nvim"], ["vim"], ["nano"], ["code", "--wait"])
command = ["nvim"]

[ui]
# Téma barev TUI: "auto", "dracula", "tokyo-night", "nord", nebo "default"
theme = "dracula"
# Styl Markdown náhledu: "auto", "dark", "light", nebo "notty"
preview_style = "dark"
# Zobrazovat název Git větve ve stavovém řádku
show_branch = true
# Šířka panelu Scopes ve sloupcích (výchozí: 24)
scope_width = 24
# Výchozí pohled po spuštění: "now", "project", "global", nebo "all"
default_view = "now"

[storage]
# Režim úložiště: "central" (ve stavovém adresáři) nebo "repo" (.herdr/logbook v projektu)
project_mode = "central"
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

Zkratka funguje jako přepínač, místo aby otevírala další a další panely: pokud je panel Logbooku otevřený, ale nemá fokus, zkratka vás do něj přepne; dalším stiskem z tohoto panelu se zavře.

Znovu načtěte konfiguraci Herdru:
```bash
herdr config check
herdr server reload-config
```

---

## 🎯 Aktuální úkol (`now`)

`now.md` drží to, na čem právě pracujete. Nastavíte ho v Hubu klávesou `t`, nebo z terminálu:

```bash
herdr-logbook now                             # vypíše aktuální úkol
herdr-logbook now "Rotace podpisových tokenů" # nastaví ho
herdr-logbook now --project-root ~/src/api "Oprava padajícího login testu"
```

Přepnutí úkolu připojí ten předchozí do měsíčního souboru v inboxu jako `Task done: …` s časovým razítkem a větví. Zbytek `now.md` zůstane nedotčený — sekce `## Next steps`, `## Blockers` a `## Context` se nepřepisují.

> Přepínače musí být před textem úkolu (`now --project-root CESTA "úkol"`) a samotný text nesmí obsahovat Markdown nadpisy — ty by sekci předčasně ukončily.

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

Návrhy, hlášení chyb a PR jsou vítány! Podívejte se do [CONTRIBUTING.md](CONTRIBUTING.md) pro lokální nastavení a instrukce k testování a do [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) pro pravidla komunity.

## 📄 Licence

[MIT](LICENSE) © 2026 Alexandr Rešetňak
