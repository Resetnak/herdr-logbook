# digest.tape — VHS script that renders the activity-digest promo clip
# (assets/digest.gif). Regenerate from the repo root:  vhs digest.tape
# Needs: vhs, bash, and go — the setup below rebuilds bin/herdr-logbook itself.
#
# The clip tells one story in ~24s: capture a note, switch the active task,
# then hit `s` and watch both land in today's square on the heatmap. Unlike
# cassette.tape this never starts Herdr — the TUI runs standalone with
# HERDR_PLUGIN_STATE_DIR pointed at a throwaway store under /tmp/hl-digest-home,
# and the project is resolved from --project-root. Nothing outside /tmp is
# touched, so there is no session to stop or plugin to unlink afterwards.
# VHS: https://github.com/charmbracelet/vhs

Output assets/digest.gif

Require bash
Require go

# Narrower than the README demo on purpose: this clip is for social timelines,
# where 1600px scales down to unreadable text on a phone.
Set Shell "bash"
Set FontSize 16
Set Width 1280
Set Height 720
Set Padding 16
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

# --- Hidden setup: throwaway store, fake project, backfilled history ---
Hide
Type `export REPO=$PWD`
Enter
# Rebuild first: recording against a stale bin/ silently ships an old UI. The
# sleep covers the build; with a cold Go cache, build once by hand beforehand.
Type `go build -ldflags "-X main.version=$(sed -n 's/^version = "\(.*\)"/\1/p' herdr-plugin.toml)" -o bin/herdr-logbook ./cmd/herdr-logbook || { echo BUILD FAILED; sleep 5; }`
Enter
Sleep 20s
Type "unset HERDR_ENV HERDR_PANE_ID HERDR_TAB_ID HERDR_WORKSPACE_ID HERDR_SOCKET_PATH HERDR_BIN_PATH"
Enter
Type "rm -rf /tmp/hl-digest-home /tmp/hl-digest-project && mkdir -p /tmp/hl-digest-home /tmp/hl-digest-project/api-gateway && export HOME=/tmp/hl-digest-home"
Enter
Type `export PATH=$REPO/bin:$PATH HERDR_PLUGIN_STATE_DIR=/tmp/hl-digest-home/.local/state/herdr/plugins/herdr-logbook PS1="> "`
Enter
Type "cd /tmp/hl-digest-project/api-gateway && git init -q && git remote add origin https://github.com/acme/api-gateway.git && git -c user.email=dev@acme.dev -c user.name=dev commit --allow-empty -q -m init && git checkout -q -b feature/token-rotation"
Enter

# Today starts with one capture and one decision; the third entry is typed
# live on camera, so the standup section grows while you watch.
Type "herdr-logbook init >/dev/null && herdr-logbook capture -text 'Postgres pool exhausted under load - raise max_connections' >/dev/null && herdr-logbook decision -title 'Use opaque refresh tokens instead of long-lived JWTs' -no-edit >/dev/null"
Enter
Type `S=$(ls -d /tmp/hl-digest-home/.local/state/herdr/plugins/herdr-logbook/store/projects/p_*) && printf '# Now\n\n## Current task\nImplement token rotation for the API gateway.\n\n## Next steps\n- [ ] Add replay detection.\n- [ ] Rotate signing keys before v2 launch.\n' > $S/now.md`
Enter

# Backfill four weeks of history. Entries land in the current monthly inbox
# file regardless of their date — the heatmap reads the "## <timestamp>"
# heading, not the filename. Each "<days-ago>:<entries>" pair sets one day's
# shade (block() steps at 1, 2, 4 and 6), weekends left blank, days 1-5 kept
# busy so the streak counter has something to count.
Type `ago() { date -v-$1d "+%Y-%m-%d" 2>/dev/null || date -d "$1 days ago" "+%Y-%m-%d"; }; msgs=("Drafted the zero-downtime rotation runbook." "Added replay detection to the auth middleware." "Benchmarked the token cache under load." "Paired on the v2 migration plan." "Triaged a flaky gateway integration test." "Reviewed the token rotation plan." "Cut the release candidate for v2." "Chased down a connection leak in the pool."); n=0; for spec in 1:3 2:6 3:2 4:4 5:1 8:2 9:5 10:3 11:1 12:4 15:2 16:6 17:3 18:2 19:4 22:1 23:3 24:2 25:5; do d=${spec%%:*}; c=${spec##*:}; for k in $(seq $c); do printf '## %s %02d:%02d\n\n%s\n\n' "$(ago $d)" $((8 + k)) $((d + k)) "${msgs[$((n % 8))]}" >> $S/inbox/$(date "+%Y-%m").md; n=$((n+1)); done; done`
Enter
Type `herdr-logbook index rebuild >/dev/null`
Enter
Type "clear"
Enter
Type "herdr-logbook tui --project-root /tmp/hl-digest-project/api-gateway"
Enter
Sleep 3000ms
Show

# --- Open the project inbox so there is a list for the new note to land in ---
Sleep 1300ms
Type "j"
Sleep 700ms
Type "l"
Sleep 1100ms

# --- c: quick capture, saved without ever leaving the TUI ---
Type "c"
Sleep 900ms
Type "Rotate JWT signing keys before the v2 launch"
Sleep 900ms
Ctrl+S
Sleep 2400ms

# --- t: switch the active task; the one it replaces is filed into the inbox ---
Type "t"
Sleep 1000ms
Type "Ship the token rotation PR"
Sleep 1000ms
Ctrl+S
Sleep 2400ms

# --- s: the payoff — heatmap, streak, and today's square counting what we
# just typed. t widens the range, y copies the standup Markdown.
Type "s"
Sleep 4800ms
Type "t"
Sleep 3200ms
Type "y"
Sleep 2800ms

# --- Hidden teardown: quit the TUI, drop the throwaway store ---
Hide
Escape
Sleep 500ms
Type "q"
Sleep 800ms
Type "rm -rf /tmp/hl-digest-home /tmp/hl-digest-project"
Enter
Sleep 1000ms
