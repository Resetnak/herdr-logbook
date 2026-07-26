# cassette.tape — VHS script that renders the README demo (assets/demo.gif).
# Regenerate from the repo root:  vhs cassette.tape
# Needs: vhs, herdr (>= 0.7), claude, bash, and the plugin binary at
#   go build -ldflags "-X main.version=<version>" -o bin/herdr-logbook ./cmd/herdr-logbook
# The whole demo runs in a throwaway Herdr instance under /tmp/hl-demo-home
# (isolated HOME: own config, plugin registry, store, and socket). Your real
# Herdr server, config, and logbook data are never touched; the demo session
# is stopped and deleted at the end of the recording.
# VHS: https://github.com/charmbracelet/vhs

Output assets/demo.gif

Require herdr
Require claude
Require bash

Set Shell "bash"
Set FontSize 15
Set Width 1600
Set Height 800
Set Padding 16
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

# --- Hidden setup: throwaway HOME for Herdr; claude keeps the real one ---
Hide
Type `export REPO=$PWD REAL_HOME=$HOME`
Enter
Type "unset HERDR_ENV HERDR_PANE_ID HERDR_TAB_ID HERDR_WORKSPACE_ID HERDR_SOCKET_PATH HERDR_BIN_PATH"
Enter
Type `unset $(env | sed -n 's/^\(CLAUDE[A-Z_0-9]*\)=.*/\1/p') 2>/dev/null; true`
Enter
Type "rm -rf /tmp/hl-demo-home /tmp/herdr-demo && mkdir -p /tmp/hl-demo-home/.config/herdr /tmp/herdr-demo/api-gateway && export HOME=/tmp/hl-demo-home"
Enter
Type `printf 'onboarding = false\n\n[terminal]\ndefault_shell = "/bin/bash"\nshell_mode = "non_login"\nnew_cwd = "/tmp/herdr-demo/api-gateway"\n\n[ui]\nsidebar_start_collapsed = true\nsidebar_collapsed_mode = "hidden"\n\n[[keys.command]]\nkey = "prefix+m"\ntype = "plugin_action"\ncommand = "herdr-logbook.open"\ndescription = "Herdr Logbook"\n\n[theme]\nname = "catppuccin"\n\n[experimental]\nallow_nested = true\n' > $HOME/.config/herdr/config.toml`
Enter
Type `printf '{"statusLine":{"type":"command","command":"echo api-gateway"}}' > /tmp/hl-demo-home/claude-settings.json`
Enter
Type `printf 'PS1="> "\nalias claude="HOME=%s claude --setting-sources project --strict-mcp-config --model haiku --settings /tmp/hl-demo-home/claude-settings.json --disallowedTools Bash,Read,Glob,Grep,WebSearch,WebFetch,Task,Edit,Write,TodoWrite,Skill,NotebookEdit --append-system-prompt %s"\nexport PATH=%s/bin:$PATH\n' "$REAL_HOME" "'Assume a typical Go API gateway. Never ask clarifying questions; write the full answer immediately in English.'" "$REPO" > /tmp/hl-demo-home/.bashrc`
Enter
Type `herdr plugin link $REPO --enabled >/dev/null 2>&1`
Enter
Type "cd /tmp/herdr-demo/api-gateway && git init -q && git remote add origin https://github.com/acme/api-gateway.git && git -c user.email=dev@acme.dev -c user.name=dev commit --allow-empty -q -m init && git checkout -q -b feature/token-rotation"
Enter
Type `export PATH=$REPO/bin:$PATH HERDR_PLUGIN_STATE_DIR=/tmp/hl-demo-home/.local/state/herdr/plugins/herdr-logbook`
Enter
Type "herdr-logbook init >/dev/null && herdr-logbook capture -text 'Rotate JWT signing keys before the v2 launch' >/dev/null && herdr-logbook capture -text 'Postgres pool exhausted under load - raise max_connections' >/dev/null && herdr-logbook decision -title 'Use opaque refresh tokens instead of long-lived JWTs' -no-edit >/dev/null"
Enter
Type `S=$(ls -d /tmp/hl-demo-home/.local/state/herdr/plugins/herdr-logbook/store/projects/p_*) && printf '# Now\n\n## Current task\nImplement token rotation for the API gateway.\n\n## Next steps\n- [ ] Add replay detection.\n- [ ] Rotate signing keys before v2 launch.\n' > $S/now.md && herdr-logbook index rebuild >/dev/null && unset HERDR_PLUGIN_STATE_DIR`
Enter
Type "clear"
Enter
Type "herdr --session demo"
Enter
Sleep 4500ms
Show

# --- Inside Herdr: ask Claude for help in the fake project ---
Sleep 800ms
Type `claude -p "Write a step-by-step guide for rotating our JWT signing keys with zero downtime."`
Sleep 500ms
Enter
Hide
Sleep 26000ms
Show
Sleep 2200ms

# --- prefix+m: the Logbook Hub splits in beside the agent ---
Ctrl+B
Sleep 300ms
Type "m"
Sleep 3500ms

# --- Browse the Project Inbox, preview a note, search everything ---
Type "j"
Sleep 900ms
Type "l"
Sleep 900ms
Type "j"
Sleep 800ms
Type "v"
Sleep 2000ms
Type "/"
Sleep 600ms
Type "jwt"
Sleep 2200ms
Escape
Sleep 1000ms

# --- t: switch the active task; the one it replaces is filed into the inbox ---
Type "t"
Sleep 1200ms
Type "Ship the token rotation PR"
Sleep 1200ms
Ctrl+S
Sleep 2800ms

# --- Hidden teardown: detach and remove the throwaway session ---
Hide
Ctrl+B
Sleep 300ms
Type "q"
Sleep 1500ms
Type "herdr session stop demo && herdr session delete demo"
Enter
Sleep 1500ms
