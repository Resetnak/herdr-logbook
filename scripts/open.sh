#!/usr/bin/env bash
set -eu

herdr_bin="${HERDR_BIN_PATH:-herdr}"
plugin_root="${HERDR_PLUGIN_ROOT:?HERDR_PLUGIN_ROOT is required}"
logbook_bin="$plugin_root/bin/herdr-logbook"
entrypoint="${1:-hub}"
placement=overlay
# Pane label reported by herdr; keep in sync with the [[panes]] titles in herdr-plugin.toml.
label="Capture Logbook"
if [ "$entrypoint" = "hub" ]; then
  placement=split
  label="Herdr Logbook"
fi

# Toggle: focus the pane this entrypoint already owns, and close it once it has focus.
# ponytail: matched on the pane label because `herdr pane list` reports no plugin id;
# an unparseable listing just falls through to opening a pane, as before.
pane_ids() { sed -n 's/.*"pane_id":"\([^"]*\)".*/\1/p'; }
open_panes="$("$herdr_bin" pane list 2>/dev/null |
  tr '{' '\n' |
  grep -F "\"label\":\"$label\"" || true)"
if [ -n "$open_panes" ]; then
  if printf '%s\n' "$open_panes" | grep -qF '"focused":true'; then
    for pane_id in $(printf '%s\n' "$open_panes" | pane_ids); do
      "$herdr_bin" plugin pane close "$pane_id" >/dev/null
    done
  else
    "$herdr_bin" plugin pane focus "$(printf '%s\n' "$open_panes" | pane_ids | head -n 1)" >/dev/null
  fi
  exit 0
fi

cwd="$("$logbook_bin" resolve-cwd)"

set -- plugin pane open \
  --plugin herdr-logbook \
  --entrypoint "$entrypoint" \
  --placement "$placement" \
  --focus

if [ "$entrypoint" = "hub" ]; then
  set -- "$@" --direction right
fi
if [ -n "$cwd" ] && [ -d "$cwd" ]; then
  set -- "$@" --cwd "$cwd"
fi

exec "$herdr_bin" "$@"
