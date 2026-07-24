# cassette.tape — VHS script that renders the README demo (assets/demo.gif).
# Regenerate:  vhs cassette.tape   (needs `vhs` plus the `herdr-logbook` binary on PATH)
# VHS: https://github.com/charmbracelet/vhs

Output assets/demo.gif

Require herdr-logbook

Set FontSize 18
Set Width 1200
Set Height 720
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

# --- Hidden setup: isolated store, seeded from a neutral demo project ---
Hide
Type "export HERDR_PLUGIN_STATE_DIR=$(mktemp -d)"
Enter
Type "rm -rf /tmp/herdr-demo && mkdir -p /tmp/herdr-demo/api-gateway && cd /tmp/herdr-demo/api-gateway"
Enter
Type "git init -q && git remote add origin https://github.com/acme/api-gateway.git && git -c user.email=dev@acme.dev -c user.name=dev commit --allow-empty -q -m init && git checkout -q -b feature/token-rotation"
Enter
Type "herdr-logbook init >/dev/null"
Enter
Type "herdr-logbook capture -text 'Rotate JWT signing keys before the v2 launch' >/dev/null"
Enter
Type "herdr-logbook capture -text 'Postgres pool exhausted under load — raise max_connections' >/dev/null"
Enter
Type "herdr-logbook index rebuild >/dev/null"
Enter
Type "clear"
Enter
Sleep 500ms
Show

# --- Commands a user runs right after installing ---
Type "herdr-logbook version"
Sleep 400ms
Enter
Sleep 1200ms

Type "herdr-logbook capture -text 'Ship the offline logbook demo'"
Sleep 400ms
Enter
Sleep 1200ms

Type "herdr-logbook doctor"
Sleep 400ms
Enter
Sleep 2200ms

# --- The interactive Hub ---
Type "herdr-logbook tui"
Sleep 400ms
Enter
Sleep 2500ms

# browse the Project Inbox, open a note, then search across projects
Type "j"
Sleep 1000ms
Type "l"
Sleep 900ms
Type "j"
Sleep 800ms
Type "v"
Sleep 2200ms
Type "/"
Sleep 600ms
Type "jwt"
Sleep 2400ms
Escape
Sleep 800ms
Type "q"
Sleep 1500ms
