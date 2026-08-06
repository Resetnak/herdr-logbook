# demo-x.tape — the 16:9 social clip (assets/herdr-logbook-x.gif / .mp4).
#
# Regenerate from the repo root:  ./scripts/render-demo-x.sh
# (or, once bin/herdr-logbook exists:  vhs assets/demo-x.tape)
#
# One story in ~15s: open the Hub in a project, see the working memory that is
# already there, record a decision, then hit `s` and watch that decision get
# counted on the activity heatmap and in today's standup. No feature tour.
#
# Like digest.tape this never starts Herdr — the Hub runs standalone with
# HERDR_PLUGIN_STATE_DIR / HERDR_PLUGIN_CONFIG_DIR pointed at the throwaway store
# that scripts/demo-x-fixture.sh builds under /tmp/hl-x. Your real Herdr server,
# config, plugin registry, and logbook data are never read or written.
#
# VHS: https://github.com/charmbracelet/vhs

Output "assets/herdr-logbook-x.gif"
Output "assets/herdr-logbook-x.mp4"

Require bash
Require git

# 1280x720 is 16:9 for the timeline, and both sides are even — h.264 refuses an
# odd height, so 1200x675 renders a GIF but no MP4. FontSize 26 leaves ~80
# columns, which is the real constraint: columns, not pixels, decide whether this
# is legible on a phone. The window is narrow at both ends — below 70 the Hub
# drops its Scopes pane and the story loses its left context, and above ~80 the
# decision title stops fitting on one row of the notes list.
Set Shell "bash"
Set FontSize 26
Set Width 1280
Set Height 720
Set Padding 12
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

# --- Hidden setup: throwaway store and fake project, nothing outside /tmp ---
Hide
Type `export REPO=$PWD PS1="> "`
Enter
Type `bash "$REPO/scripts/demo-x-fixture.sh" >/dev/null`
Enter
Sleep 2s
Type `export PATH="$REPO/bin:$PATH" HERDR_PLUGIN_STATE_DIR=/tmp/hl-x/state HERDR_PLUGIN_CONFIG_DIR=/tmp/hl-x/config`
Enter
Type "cd /tmp/hl-x/project/api-gateway && clear"
Enter
Show

# --- Open the Hub. Same binary Herdr runs for the `open` action / prefix+m ---
Sleep 600ms
Type "herdr-logbook tui"
Sleep 300ms
Enter
Sleep 1800ms

# --- The memory that is already here: notes, a decision, and now.md ---
Sleep 1300ms

# --- d: record a decision. It saves inside the TUI, no editor round trip ---
Type "d"
Sleep 800ms
Type "Keep notes local and Markdown-first"
Sleep 600ms
Ctrl+S

# --- ...and it is in the list. Plain Markdown on disk, no database ---
Sleep 1900ms

# --- s: four weeks of activity, and a streak. The fixture backfills the history,
# so the heatmap has something to show instead of one lonely square.
Type "s"
Sleep 2400ms

# --- The standup is below the fold: PageDown clamps to the bottom, where the
# decision from ten seconds ago is already counted, under the active task.
PageDown
Sleep 2600ms

# --- Hidden teardown: close the digest, quit the Hub, drop the throwaway store ---
Hide
Escape
Sleep 300ms
Type "q"
Sleep 600ms
Type "rm -rf /tmp/hl-x"
Enter
Sleep 800ms
