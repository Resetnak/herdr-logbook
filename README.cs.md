<div align="center">

# 📓 Herdr Logbook

**Lokální, offline pracovní paměť pro vývojáře v Herdru — postavená na Markdownu.**

Drží aktivní úkol, rychlé poznámky, technická rozhodnutí a projektové zápisky přímo u terminálu — žádný proprietární formát, žádný cloud, žádná telemetrie.

<br>

[![CI](https://github.com/Resetnak/herdr-logbook/actions/workflows/ci.yml/badge.svg)](https://github.com/Resetnak/herdr-logbook/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![Licence: MIT](https://img.shields.io/badge/Licence-MIT-green.svg)](LICENSE)
[![Platformy](https://img.shields.io/badge/platformy-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)](docs/herdr-compatibility.md)
[![Herdr](https://img.shields.io/badge/Herdr-%E2%89%A50.7.0-2088FF)](herdr-plugin.toml)
[![Stav](https://img.shields.io/badge/stav-pre--release-orange)](CHANGELOG.md)

<br>

[English](README.md) · **Čeština**

[Proč](#proč-tenhle-plugin-dělám) · [Instalace](#instalace) · [Příkazy](#příkazy) · [Úložiště](#úložiště-a-rozpoznání-projektu) · [Konfigurace](#konfigurace) · [Soukromí](#soukromí-a-bezpečnost-dat) · [Přispívání](CONTRIBUTING.md)

</div>

---

Herdr Logbook je lokální pracovní deník postavený na Markdownu pro vývojáře používající Herdr. Drží aktivní úkol, rychlé poznámky, technická rozhodnutí a projektové zápisky u terminálu, aniž by zaváděl proprietární formát dokumentů.

Základní smyčka je záměrně malá: otevři Hub pro aktuální repozitář, zachyť kontext dřív, než zmizí, prohlédni si Markdown, prohledej registrované projekty a pokračuj v úpravách ve svém běžném editoru.

## Proč tenhle plugin dělám

Užitečný kontext kolem vývojového úkolu často přežije terminálovou relaci, ve které vznikl, ale nepotřebuje kvůli tomu další cloudovou službu ani proprietární databázi. Herdr Logbook dělám proto, aby byl tento kontext v Herdru dostupný na jednu zkratku: lokálně, prohledávatelně a v Markdownu, který zůstane použitelný i bez pluginu.

```text
┌ Scopes ─────────┬ Notes ─────────────────────┬ Preview ───────────────────────┐
│ ● Now           │ now.md                     │ # Now                         │
│ Project Inbox   │ auth-notes.md              │ ## Current task               │
│ Project Notes   │ use-redis.md               │ Implement token rotation.     │
│ Decisions       │                            │                               │
│ Global Inbox    │                            │ ## Next steps                  │
│ All Notes       │                            │ - [ ] Add replay detection.   │
└─────────────────┴────────────────────────────┴────────────────────────────────┘
 api-gateway · feature/token-rotation · central store · / search · ? help
```

## Hlavní vlastnosti

- **Markdown je zdroj pravdy.** Každá poznámka je prostý soubor `.md`, který vlastníš. Vyhledávací cache je jednorázová a sama se obnovuje.
- **Vědomí projektu.** Poznámky jsou vázané na repozitář, ve kterém právě jsi — rozpoznaný přes kontext Herdru, Git worktrees nebo aktuální adresář.
- **Offline a soukromé.** Žádná telemetrie, síťová volání, cloud sync, AI, automatické Git operace ani spouštění příkazů nalezených v poznámkách.
- **Responzivní TUI.** Bubble Tea Hub se přizpůsobí ze tří panelů na jeden pod 70 sloupci a nikdy neblokuje první vykreslení skenem indexu.
- **Vlastní editor.** Plná editace je delegována na `$EDITOR`; TUI neimplementuje vlastní textový editor.

## Postupy práce

- `c` a `C` zachytí poznámku do měsíční schránky projektu nebo globální.
- `n` vytvoří projektovou poznámku; `d` vytvoří datovaný záznam rozhodnutí.
- `/` prohledá jednorázový index napříč registrovanými úložišti projektů.
- `e` uspí Hub a otevře vybraný Markdown soubor nakonfigurovaným editorem.
- `now.md` se vytvoří jednou a v pohledu projektu je vždy první.

Hub používá Bubble Tea a přizpůsobí se ze tří panelů při 110 sloupcích na jeden aktivní panel pod 70 sloupci. Náhled Markdownu se vykresluje v procesu přes Glamour; externí binárka `glow` není potřeba.

## Instalace

Herdr Logbook se aktuálně instaluje ze zdrojového kódu. Potřebuješ Herdr 0.7.0 nebo novější, Git a Go 1.25 nebo novější (viz [`go.mod`](go.mod)).

### macOS, Linux a WSL

```bash
git clone https://github.com/Resetnak/herdr-logbook.git
cd herdr-logbook
mkdir -p bin
go build -o bin/herdr-logbook ./cmd/herdr-logbook
herdr plugin link "$(pwd)" --enabled
herdr plugin action invoke open --plugin herdr-logbook
```

### Windows PowerShell

```powershell
git clone https://github.com/Resetnak/herdr-logbook.git
Set-Location herdr-logbook
New-Item -ItemType Directory -Force bin | Out-Null
go build -o bin\herdr-logbook.exe .\cmd\herdr-logbook
herdr plugin link (Get-Location) --enabled
herdr plugin action invoke open-windows --plugin herdr-logbook
```

`herdr plugin link` nespouští build. Nejdřív sestav binárku.

Po zveřejnění tagované verze ji Herdr umí nainstalovat přímo:

```bash
herdr plugin install Resetnak/herdr-logbook --ref vX.Y.Z -y
```

Balení releasu je nakonfigurováno pro Linux, macOS a Windows na amd64 a arm64. Instalátory stáhnou přesnou verzi z manifestu a ověří `checksums.txt`. Veřejný release `v0.1.0` nesmí být deklarován, dokud není kompletní ruční matice kompatibility v [docs/herdr-compatibility.md](docs/herdr-compatibility.md).

## Příkazy

```text
herdr-logbook tui --view now|project|global|all [--project-root PATH] [--editor CMD]
herdr-logbook capture --scope project|global [--text TEXT | --stdin | --selected]
herdr-logbook decision [--title TEXT] [--project-root PATH]
herdr-logbook init --storage central|repo [--project-root PATH]
herdr-logbook doctor [--json] [--project-root PATH]
herdr-logbook keybinds
herdr-logbook paths [--json] [--project-root PATH]
herdr-logbook index rebuild [--project-root PATH]
herdr-logbook version
```

Bez zdroje pro zachycení otevře `capture` textové UI. Bez `--title` se `decision` zeptá na název. Vytvoření rozhodnutí otevře nakonfigurovaný externí editor, pokud se pro automatizaci nepoužije `--no-edit`.

Návratové kódy mají význam: `2` použití, `3` rozpoznání kontextu/stavu, `4` zámek/zápis úložiště, `5` kontext Herdru, `6` editor.

## Úložiště a rozpoznání projektu

Výchozí je centrální úložiště a repozitáře nechává netknuté:

```text
$HERDR_PLUGIN_STATE_DIR/
├── store/projects/p_<sha256>/{now.md,inbox,notes,decisions}
├── store/global/{inbox,notes,decisions}
├── registry/projects.toml
└── cache/index-v1.json
```

`init --storage repo` výslovně zapíná `.herdr/logbook/` a vypíše doporučené pravidlo pro ignorování; `.gitignore` nikdy neupravuje. Inicializace zachová každý existující neprázdný soubor. Cache soubory jsou jednorázové a obnoví se z Markdownu příkazem `index rebuild`.

Priorita kontextu je explicitní `--project-root`, cesta Herdr worktree, cwd zaměřeného panelu, cwd workspace, `--cwd`, pak cwd procesu. Git worktrees sdílejí identitu přes otisk remotu bez přihlašovacích údajů nebo přes společný Git adresář. Ne-Git projekty používají kanonickou cestu. `.herdr-logbook.toml` může vybrat podprojekt v monorepu, ale nemůže z repozitáře uniknout přes traverzování cest ani symlinky.

## Konfigurace

Herdr a Herdr Logbook používají dva oddělené konfigurační soubory.

### Výběr editoru

Adresář konfigurace pluginu zjistíš příkazem `herdr plugin config-dir herdr-logbook`. V něm vytvoř nebo uprav `config.toml`:

```toml
[editor]
command = ["nvim"]
```

Argumenty zapisuj jako samostatné položky pole, například `command = ["code", "--wait"]`. Pořadí editoru je `tui --editor CMD`, `editor.command`, `$HERDR_LOGBOOK_EDITOR`, `$VISUAL`, `$EDITOR`, pak výchozí hodnoty platformy. Příkazy se spouštějí jako argv, nikdy se neinterpolují přes shell.

### Zkratka v Herdru

Do `~/.config/herdr/config.toml` přidej akci pluginu:

```toml
[[keys.command]]
key = "prefix+m"
type = "plugin_action"
command = "herdr-logbook.open"
description = "Herdr Logbook"
```

Na Windows použij `herdr-logbook.open-windows`. Soubor ověř a znovu načti:

```bash
herdr config check
herdr server reload-config
```

`--editor` patří binárce Logbooku, takže funguje při přímém spuštění, například `herdr-logbook tui --editor nvim`. Nepřidávej ho za `herdr plugin action invoke`; Herdr argumenty akce nepředává. Pokud chceš editor změnit jen pro jednu zkratku Herdru, předej ho panelu přes proměnnou prostředí:

```toml
[[keys.command]]
key = "prefix+m"
type = "shell"
command = "herdr plugin pane open --plugin herdr-logbook --entrypoint hub --placement split --direction right --focus --env HERDR_LOGBOOK_EDITOR=nvim"
description = "Herdr Logbook"
```

Neznámé konfigurační klíče vyvolají varování. Neplatné typy a hodnoty selžou explicitně. Uživatelská konfigurace se během kompatibilních migrací nepřepisuje.

## Soukromí a bezpečnost dat

Běžný běh nemá žádnou telemetrii, síťové požadavky, cloud sync, AI volání, automatické Git operace ani spouštění příkazů z poznámek. Vybraný text terminálu se nikdy neobjeví v diagnostice. Git přihlašovací údaje se odstraní před výstupem do registru a diagnostiky. Zachycení používá omezené zámky, dočasné soubory ve stejném adresáři, sync souborového systému a atomické nahrazení.

Index čte pouze Markdown pod známými kořeny paměti, přeskakuje skryté adresáře a symlinky a sám se opraví po poškození cache. Odebrání pluginu neodstraní Markdown data.

## Omezení a plán

Herdr Logbook není obecný systém pro správu znalostí, kolaborační služba, synchronizační engine, prohlížečové UI ani vlastní textový editor. Archiv/koš, uložená hledání, šablony, backlinky a volitelná integrace externího Glow jsou odloženy, dokud nebude mít základní workflow reálné důkazy o používání. AI, cloud sync a vizualizace grafu jsou výslovné ne-cíle.

Snímky obrazovky nebo záznam terminálu budou přidány až po úspěšném průchodu maticí kompatibility na reálných hostech.

## Dokumentace

- [CHANGELOG.md](CHANGELOG.md) — poznámky k verzím.
- [SECURITY.md](SECURITY.md) — hlášení zranitelností a bezpečnostní model za běhu.
- [CONTRIBUTING.md](CONTRIBUTING.md) — vývojový postup a kroky ověření.
- [docs/herdr-compatibility.md](docs/herdr-compatibility.md) — matice kompatibility na reálných hostech.

## Licence

[MIT](LICENSE) © 2026 Alexandr Rešetňak
