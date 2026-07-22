# Logbook Right-Side Split Implementation Plan

**For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Open the macOS/Linux Logbook Hub as a focused right-side Herdr split while leaving Capture as an overlay.

**Architecture:** Keep Herdr responsible for pane layout. The existing launcher chooses native placement from the requested entrypoint; the manifest supplies the same Hub default. No post-open resize or custom pane lifecycle is added.

**Tech Stack:** Bash, Herdr plugin manifest TOML, existing shell and Go verification.

## Global Constraints

- The `hub` entrypoint uses `split` with direction `right`.
- The `capture` entrypoint remains `overlay`.
- Accept Herdr's native split ratio; do not add open-then-resize behavior.
- Preserve current working-directory resolution, focus, and exit propagation.
- Leave Windows launch behavior unchanged.

---

### Task 1: Select Native Placement by Entrypoint

**Files:**
- Create: `scripts/open_test.sh`
- Modify: `scripts/open.sh`
- Modify: `herdr-plugin.toml`

- [x] **Step 1: Write the failing launcher test**

Create `scripts/open_test.sh` with temporary stub binaries. Run `scripts/open.sh hub` and `scripts/open.sh capture`, record the Herdr arguments, and assert these exact argument strings:

```text
plugin pane open --plugin herdr-logbook --entrypoint hub --placement split --focus --direction right
plugin pane open --plugin herdr-logbook --entrypoint capture --placement overlay --focus
```

- [x] **Step 2: Run the launcher test and verify RED**

Run:

```bash
rtk proxy bash scripts/open_test.sh
```

Expected: FAIL because the current Hub arguments contain `--placement overlay` and no `--direction right`.

- [x] **Step 3: Implement the minimal launcher branch**

In `scripts/open.sh`, choose placement before constructing arguments and append direction only for Hub:

```bash
placement=overlay
if [ "$entrypoint" = "hub" ]; then
  placement=split
fi

set -- plugin pane open \
  --plugin herdr-logbook \
  --entrypoint "$entrypoint" \
  --placement "$placement" \
  --focus

if [ "$entrypoint" = "hub" ]; then
  set -- "$@" --direction right
fi
```

Change only the `hub` pane declaration in `herdr-plugin.toml`:

```toml
placement = "split"
```

- [x] **Step 4: Run the focused test and verify GREEN**

Run:

```bash
rtk proxy bash scripts/open_test.sh
rtk proxy bash -n scripts/open.sh scripts/open_test.sh scripts/install.sh
```

Expected: both commands exit 0.

- [x] **Step 5: Run project verification and rebuild the linked plugin binary**

Run:

```bash
rtk test env GOCACHE=/tmp/herdr-logbook-go-cache go test ./...
rtk test env GOCACHE=/tmp/herdr-logbook-go-cache go test -race ./...
rtk proxy env GOCACHE=/tmp/herdr-logbook-go-cache go vet ./...
rtk proxy env GOCACHE=/tmp/herdr-logbook-go-cache go build -o bin/herdr-logbook ./cmd/herdr-logbook
rtk git diff --check
```

Expected: all commands exit 0. The rebuilt ignored binary makes the locally linked Herdr plugin use the current source.
