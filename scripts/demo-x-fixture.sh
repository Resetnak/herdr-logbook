#!/usr/bin/env bash
# demo-x-fixture.sh — builds the throwaway project and store that assets/demo-x.tape
# records. Idempotent: it wipes /tmp/hl-x and rebuilds it from scratch every run, so
# two recordings a week apart show the same screen.
#
# Nothing outside /tmp/hl-x is touched. No ~/.config/herdr, no Herdr server, no
# plugin registry — the Hub runs standalone against HERDR_PLUGIN_STATE_DIR.
set -euo pipefail

ROOT=/tmp/hl-x
STATE=$ROOT/state
CONFIG=$ROOT/config
PROJECT=$ROOT/project/api-gateway
BIN=${BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/bin/herdr-logbook}

[ -x "$BIN" ] || { echo "demo-x-fixture: missing binary $BIN (go build -o bin/herdr-logbook ./cmd/herdr-logbook)" >&2; exit 1; }

rm -rf "$ROOT"
mkdir -p "$STATE" "$CONFIG" "$PROJECT"

# A fake remote gives the project a stable identity hash, so the store path is the
# same on every run and the status bar shows a name instead of a temp directory.
git -C "$PROJECT" init -q
git -C "$PROJECT" remote add origin https://github.com/acme/api-gateway.git
git -C "$PROJECT" -c user.email=dev@acme.dev -c user.name=dev commit --allow-empty -q -m init
# Short branch name on purpose: the status bar is one line, and at ~74 columns
# "feature/…" pushes "? help" off the end once a flash message shares the row.
git -C "$PROJECT" checkout -q -b feat/token-rotation

# 20 is the narrowest Scopes pane that still fits "Project Notes" on one line, and
# every column it gives back goes to note titles — at ~80 columns (what a 1280px
# canvas leaves once the font is big enough to survive a phone screen) a wrapped
# title costs a whole row. default_view = "all" opens on the one list that holds
# notes, the decision, and now.md together.
cat > "$CONFIG/config.toml" <<'TOML'
version = 1

[ui]
theme = "dracula"
preview_style = "dark"
show_branch = true
scope_width = 20
default_view = "all"

[storage]
project_mode = "central"
TOML

export HERDR_PLUGIN_STATE_DIR=$STATE HERDR_PLUGIN_CONFIG_DIR=$CONFIG

hl() { "$BIN" "$@" --project-root "$PROJECT"; }

hl init >/dev/null
# Short enough that the standup renders it as "• <text> (api-gateway)" on one row;
# the digest viewport truncates at the right edge rather than wrapping.
hl capture -text 'Postgres pool exhausted under load' >/dev/null
# Titles are kept short enough to render on one row in the notes pane; a wrapped
# title reads as a layout bug in a clip this small.
hl decision -title 'Use opaque refresh tokens' -no-edit >/dev/null

S=$(ls -d "$STATE"/store/projects/p_*)

cat > "$S/now.md" <<'MD'
# Now

## Current task
Implement token rotation for the API gateway.

## Next steps
- [ ] Add replay detection.
- [ ] Rotate signing keys before the v2 launch.
MD

cat > "$S/notes/token-rotation-design.md" <<'MD'
# Token rotation design

Refresh tokens are rotated on every use; the old one is revoked in the same
transaction so a replayed token is always detectable.
MD

cat > "$S/notes/gateway-runbook.md" <<'MD'
# Gateway runbook

Zero-downtime key rollout: publish the new JWK, wait one refresh window, then
retire the old signing key.
MD

cat > "$S/notes/rate-limit-tuning.md" <<'MD'
# Rate limit tuning

Per-tenant buckets, not per-IP: shared NAT egress was eating the whole quota
for the mobile client.
MD

# Four weeks of backfill, so pressing `s` lands on a heatmap with something in it
# instead of one lonely square. Entries go into the current monthly inbox file
# whatever their date — the heatmap reads the "## <timestamp>" heading, not the
# filename. Each "<days-ago>:<entries>" pair sets one day's shade (the renderer
# steps at 1, 2, 4 and 6 entries), weekends are left blank, and days 1-5 are kept
# busy so the streak counter has a run to count. Dates are relative to today, so
# the squares move with the calendar but the *shape* is identical on every run.
ago() { date -v-"$1"d "+%Y-%m-%d" 2>/dev/null || date -d "$1 days ago" "+%Y-%m-%d"; }
msgs=(
	"Drafted the zero-downtime rotation runbook."
	"Added replay detection to the auth middleware."
	"Benchmarked the token cache under load."
	"Paired on the v2 migration plan."
	"Triaged a flaky gateway integration test."
	"Reviewed the token rotation plan."
	"Cut the release candidate for v2."
	"Chased down a connection leak in the pool."
)
n=0
for spec in 1:3 2:6 3:2 4:4 5:1 8:2 9:5 10:3 11:1 12:4 15:2 16:6 17:3 18:2 19:4 22:1 23:3 24:2 25:5; do
	day=${spec%%:*}
	count=${spec##*:}
	for k in $(seq "$count"); do
		printf '## %s %02d:%02d\n\n%s\n\n' \
			"$(ago "$day")" $((8 + k)) $((day + k)) "${msgs[$((n % 8))]}" \
			>> "$S/inbox/$(date "+%Y-%m").md"
		n=$((n + 1))
	done
done

"$BIN" index rebuild --project-root "$PROJECT" >/dev/null

echo "demo-x-fixture: ready at $PROJECT (store $S)"
