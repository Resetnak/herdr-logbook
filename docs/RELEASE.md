# Release checklist

How to cut a public release of Herdr Logbook. The repository is currently
**private**; going public and publishing a release is a single, deliberate
sequence — do it top to bottom, don't stop halfway.

Herdr itself is a public tool: <https://herdr.dev> (`brew install herdr`,
source at <https://github.com/ogulcancelik/herdr>). Herdr Logbook is a plugin
for it.

## 1. Pre-flight (while still private)

- [ ] `go test ./...` and `go test -race ./...` pass.
- [ ] `go vet ./...`, `staticcheck ./...`, `govulncheck ./...` clean.
- [ ] `gofmt -l .` prints nothing.
- [ ] Version is consistent: `herdr-plugin.toml` `version`, `CHANGELOG.md`
      top entry, and the tag you are about to push all match.
- [ ] `CHANGELOG.md` has a dated entry for this version (move items out of
      `Unreleased`).
- [ ] README links resolve (no `github.com/Resetnak/herdr` — Herdr lives at
      `herdr.dev`). English and Czech READMEs in sync.
- [ ] Demo GIF is current: `vhs cassette.tape` regenerates `assets/demo.gif`
      (needs `vhs` + the `herdr-logbook` binary on `PATH`).
- [ ] Plugin integration verified on the host you claim in
      `docs/herdr-compatibility.md`:
      ```bash
      go build -ldflags "-X main.version=$(awk -F '"' '/^version = /{print $2;exit}' herdr-plugin.toml)" -o bin/herdr-logbook ./cmd/herdr-logbook
      herdr plugin link "$(pwd)" --enabled          # link does NOT build — build first
      herdr plugin action invoke open --plugin herdr-logbook     # Hub opens, exit 0
      herdr plugin action invoke capture --plugin herdr-logbook  # capture works
      ```
      Only claim the platforms you actually exercised. Cross-compiled ≠ verified.

## 2. Go public

The install scripts fetch release assets with an unauthenticated `curl`, so
they only work once the repo is public. Nothing in the install/`herdr plugin
install` path can be tested while private.

```bash
gh repo edit Resetnak/herdr-logbook --visibility public --accept-visibility-change-consequences
```

## 3. Tag and release

Pushing a `v*` tag triggers `.github/workflows/release.yml` → GoReleaser
builds linux/darwin/windows × amd64/arm64, publishes archives + `checksums.txt`.

```bash
git tag v0.0.2            # must equal herdr-plugin.toml version
git push origin v0.0.2
```

- [ ] GoReleaser workflow succeeds.
- [ ] Release page lists all six archives and `checksums.txt`.
- [ ] CodeQL workflow now runs (it self-skips on private repos).

## 4. Post-release smoke test (do it within minutes)

- [ ] Script installer downloads and verifies the checksum:
      ```bash
      curl -fsSL https://raw.githubusercontent.com/Resetnak/herdr-logbook/main/scripts/install.sh | bash
      ```
- [ ] Windows path: `irm .../install.ps1 | iex` on a Windows host (if claimed).
- [ ] Plugin-manager path resolves the release binary:
      ```bash
      herdr plugin install Resetnak/herdr-logbook
      ```
- [ ] Open the Hub once through Herdr and capture a note end to end.

If anything 404s or fails checksum, fix and re-tag before announcing.

## 5. Announce & distribution

Shipping the release gets you a page; distribution gets you users. Do these
once, right after the smoke test passes.

Repo storefront (one-time, right after going public):

- [ ] Description + topics:
      ```bash
      gh repo edit Resetnak/herdr-logbook \
        --description "Your terminal's working memory — offline, Markdown-first notes, decisions, and an active-task now.md for Herdr" \
        --add-topic go --add-topic tui --add-topic bubbletea --add-topic cli \
        --add-topic markdown --add-topic note-taking --add-topic adr \
        --add-topic offline-first --add-topic herdr
      ```
- [ ] Upload a social preview image (Settings → General → Social preview) —
      link previews on HN/Reddit/X use it; a frame from the demo GIF works.
- [ ] Enable GitHub Discussions for Q&A so issues stay actionable bug reports.
- [ ] Prime Go Report Card (badge material): open
      <https://goreportcard.com/report/github.com/Resetnak/herdr-logbook> once.

Community lists (PRs / submissions):

- [ ] Community plugin list: <https://github.com/yigitkonur/awesome-herdr>.
- [ ] Bubble Tea "in the wild" list in <https://github.com/charmbracelet/bubbletea>.
- [ ] <https://github.com/rothgar/awesome-tuis> and <https://terminaltrove.com/new/>.

Launch posts (lead with the demo GIF and the "offline, Markdown, yours" hook;
be upfront about macOS-only verification — it invites Linux testers):

- [ ] r/golang, r/commandline.
- [ ] Show HN — weekday morning US time; stay around to answer comments.

## 6. Later (after the first release settles)

- [ ] Homebrew tap: create `Resetnak/homebrew-tap`, add a `brews:` section to
      `.goreleaser.yaml` with a PAT secret (the default `GITHUB_TOKEN` cannot
      push to another repo), so `brew install resetnak/tap/herdr-logbook` works.
- [ ] Recruit platform verification from issue reports and update
      `docs/herdr-compatibility.md` as real-host reports come in.
